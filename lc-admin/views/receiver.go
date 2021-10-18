/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/driskell/log-courier/lc-admin/lcwidgets"
	"github.com/driskell/log-courier/lc-lib/admin"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type receiverResponseListener struct {
	Listen             string `json:"listen"`
	MaxPendingPayloads int    `json:"maxPendingPayloads"`
}

type receiverResponse struct {
	Connections []struct {
		CompletedLines  int    `json:"completedLines"`
		PendingPayloads int    `json:"pendingPayloads"`
		Listener        string `json:"listener"`
		Remote          string `json:"remote"`
		Description     string `json:"description"`
	} `json:"connections"`
	Listeners []*receiverResponseListener `json:"listeners"`
	Status    struct {
		ActiveConnections int `json:"activeConnections"`
	} `json:"status"`
}

// Receiver is a screen for monitoring connections in the receiver
type Receiver struct {
	*view
	client     *admin.Client
	updateChan chan<- interface{}
	data       *receiverResponse
	err        error
	table      *lcwidgets.Table
	gauges     []*widgets.Gauge
}

// NewReceiver creates a new drawable Receiver view
func NewReceiver(client *admin.Client, updateChan chan<- interface{}) View {
	p := &Receiver{
		client:     client,
		updateChan: updateChan,
	}

	p.view = newView()
	p.table = lcwidgets.NewTable()
	p.table.ColumnNames = []string{"Remote", "Description", "Lines", "Pending"}

	return p
}

// ScrollUp moves the viewable area upwards one row
func (p *Receiver) ScrollUp() {
	p.table.ScrollUp()
}

// PageUp moves the viewable area upwards one page
func (p *Receiver) PageUp() {
	p.table.PageUp()
}

// ScrollDown moves the viewable area downwards one row
func (p *Receiver) ScrollDown() {
	p.table.ScrollDown()
}

// PageDown moves the viewable area downwards one page
func (p *Receiver) PageDown() {
	p.table.PageDown()
}

// StartUpdate begins a background update, and returns result on the update channel
func (p *Receiver) StartUpdate() {
	var resp receiverResponse
	if err := p.client.RequestJSON("receiver", &resp); err != nil {
		p.updateChan <- err
		return
	}
	p.updateChan <- &resp
}

// CompleteUpdate completes the update, after which a render will occur
func (p *Receiver) CompleteUpdate(resp interface{}) {
	if err, ok := resp.(error); ok {
		p.data = nil
		p.gauges = nil
		p.err = err
		return
	}

	p.data = resp.(*receiverResponse)

	// Generate gauges
	if len(p.gauges) != len(p.data.Connections) {
		oldGauges := p.gauges
		p.gauges = make([]*widgets.Gauge, len(p.data.Connections))
		copy(p.gauges, oldGauges)
	}
	listenerIndex := make(map[string]*receiverResponseListener)
	for _, listener := range p.data.Listeners {
		listenerIndex[listener.Listen] = listener
	}
	for idx, data := range p.data.Connections {
		listener, ok := listenerIndex[data.Listener]
		if !ok {
			p.gauges[idx] = nil
			continue
		}
		if p.gauges[idx] == nil {
			p.gauges[idx] = widgets.NewGauge()
		}
		if listener.MaxPendingPayloads == 0 {
			// Support older clients
			listener.MaxPendingPayloads = 10
		}
		p.gauges[idx].Percent = int(data.PendingPayloads * 100 / listener.MaxPendingPayloads)
		p.gauges[idx].Border = false
		p.gauges[idx].Label = fmt.Sprintf("%d/%d", data.PendingPayloads, 10)
	}

	var rows [][]interface{}
	if p.data == nil {
		rows = make([][]interface{}, 1)
	} else {
		rows = make([][]interface{}, len(p.data.Connections))
	}

	if p.data == nil {
		rows[0] = []interface{}{"Loading...", "", "", ""}
	} else {
		idx := 0
		for dataIdx, data := range p.data.Connections {
			rows[idx] = make([]interface{}, 4)
			rows[idx][0] = data.Remote
			rows[idx][1] = data.Description
			rows[idx][2] = fmt.Sprintf("%d", data.CompletedLines)
			if p.gauges[dataIdx] == nil {
				rows[idx][3] = "-"
			} else {
				rows[idx][3] = p.gauges[dataIdx]
			}
			idx += 1
		}

		sort.Slice(rows, func(i, j int) bool {
			return strings.Compare(rows[i][0].(string), rows[j][0].(string)) == -1
		})
	}

	p.table.Rows = rows
}

// SetRect implements the Drawable interface
func (p *Receiver) SetRect(x1, y1, x2, y2 int) {
	p.view.SetRect(x1, y1, x2, y2)

	// 3*3+2 for dividers and padding
	// 20 for lines
	// divide amongst remaining 3 columns
	calculatedWidth := int((p.Inner.Dx() - 11 - 20) / 3)
	columnWidths := []int{calculatedWidth, calculatedWidth, 20, calculatedWidth}

	p.table.ColumnWidths = columnWidths
	p.table.SetRect(p.Min.X, p.Min.Y, p.Max.X, p.Max.Y)
}

// Draw implements the Drawable interface
func (p *Receiver) Draw(buf *ui.Buffer) {
	p.view.Draw(buf)
	if p.err != nil {
		return
	}

	p.table.Draw(buf)
}
