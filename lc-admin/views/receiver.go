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

type receiverResponse struct {
	Connections []struct {
		CompletedLines int      `json:"completedLines"`
		Speed          *float64 `json:"speed"`
		WindowBytes    int64    `json:"windowBytes"`
		WindowSize     int64    `json:"windowSize"`
		Remote         string   `json:"remote"`
		Description    string   `json:"description"`
		Status         string   `json:"status"`
		Error          string   `json:"error"`
	} `json:"connections"`
	Status struct {
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
	p.table.ColumnNames = []string{"Remote", "Description", "Lines", "Events/s", "Window"}

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

	sort.SliceStable(resp.Connections, func(i int, j int) bool {
		return strings.Compare(resp.Connections[i].Remote, resp.Connections[j].Remote) < 0
	})

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
	for idx, data := range p.data.Connections {
		// A receive window is only reported for a connection that has sent events, and is not shown once it has gone
		if data.WindowSize == 0 || data.Status == "closed" || data.Status == "failed" {
			p.gauges[idx] = nil
			continue
		}
		if p.gauges[idx] == nil {
			p.gauges[idx] = widgets.NewGauge()
		}
		p.gauges[idx].Percent = int(min(data.WindowBytes*100/data.WindowSize, 100))
		p.gauges[idx].Border = false
		p.gauges[idx].BarColor = ui.Theme.Gauge.Bar
		if p.gauges[idx].Percent >= receiverWindowWarnPercent {
			p.gauges[idx].BarColor = ui.ColorYellow
		}
	}

	var rows [][]interface{}
	if p.data == nil {
		rows = make([][]interface{}, 1)
	} else {
		rows = make([][]interface{}, len(p.data.Connections))
	}

	if p.data == nil {
		rows[0] = []interface{}{"Loading...", "", "", "", ""}
	} else {
		idx := 0
		for dataIdx, data := range p.data.Connections {
			colour := receiverStatusColour(data.Status)
			rows[idx] = make([]interface{}, 5)
			rows[idx][0] = styleCell(data.Remote, colour)
			rows[idx][1] = styleCell(data.Description, colour)
			rows[idx][2] = styleCell(fmt.Sprintf("%d", data.CompletedLines), colour)
			rows[idx][3] = styleCell(formatSpeed(data.Speed), colour)
			switch {
			case data.Status == "failed":
				rows[idx][4] = styleCell("failed: "+data.Error, colour)
			case data.Status == "closed":
				rows[idx][4] = styleCell("closed", colour)
			case p.gauges[dataIdx] == nil:
				rows[idx][4] = "-"
			default:
				rows[idx][4] = p.gauges[dataIdx]
			}
			idx += 1
		}
	}

	p.table.Rows = rows
}

// SetRect implements the Drawable interface
func (p *Receiver) SetRect(x1, y1, x2, y2 int) {
	p.view.SetRect(x1, y1, x2, y2)

	// 4*3+2 for dividers and padding
	// 20 for lines
	// 12 for events per second
	// divide amongst remaining 3 columns
	calculatedWidth := int((p.Inner.Dx() - 14 - 20 - 12) / 3)
	columnWidths := []int{calculatedWidth, calculatedWidth, 20, 12, calculatedWidth}

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

// receiverWindowWarnPercent is how full a receive window is before its gauge is shown in yellow
const receiverWindowWarnPercent = 90

// receiverStatusColour returns the colour a connection with the given status is shown in, or empty for the default
func receiverStatusColour(status string) string {
	switch status {
	case "closing", "closed":
		return "yellow"
	case "failed":
		return "red"
	}
	return ""
}

// styleCell colours a cell's text, replacing square brackets as the table would read them as styling
func styleCell(text string, colour string) string {
	if colour == "" {
		return text
	}
	return "[" + strings.NewReplacer("[", "(", "]", ")").Replace(text) + "](fg:" + colour + ")"
}
