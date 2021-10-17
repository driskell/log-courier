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
	"image"
	"sort"
	"strings"

	"github.com/driskell/log-courier/lc-admin/lcwidgets"
	"github.com/driskell/log-courier/lc-lib/admin"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type publisherResponse struct {
	Endpoints []struct {
		AverageLatency  float64 `json:"averageLatency"`
		LastError       *string `json:"last_error"`
		LastErrorTime   *string `json:"last_error_time"`
		PendingPayloads int     `json:"pendingPayloads"`
		PublishedLines  int     `json:"publishedLines"`
		Server          string  `json:"server"`
		Status          string  `json:"status"`
	} `json:"endpoints"`
	Status struct {
		PendingPayloads    int     `json:"pendingPayloads"`
		MaxPendingPayloads int     `json:"maxPendingPayloads"`
		PublishedLines     int     `json:"publishedLines"`
		Speed              float64 `json:"speed"`
	} `json:"status"`
}

type Publisher struct {
	*view
	client     *admin.Client
	updateChan chan<- interface{}
	data       *publisherResponse
	err        error
	table      *lcwidgets.Table
	gauges     []*widgets.Gauge
	mainGauge  *widgets.Gauge
}

func NewPublisher(client *admin.Client, updateChan chan<- interface{}) View {
	p := &Publisher{
		client:     client,
		updateChan: updateChan,
	}

	p.view = newView()
	p.table = lcwidgets.NewTable()
	p.mainGauge = widgets.NewGauge()

	return p
}

// StartUpdate begins a background update, and returns result on the update channel
func (p *Publisher) StartUpdate() {
	var resp publisherResponse
	if err := p.client.RequestJSON("publisher", &resp); err != nil {
		p.updateChan <- err
		return
	}
	p.updateChan <- &resp
}

// CompleteUpdate completes the update, after which a render will occur
func (p *Publisher) CompleteUpdate(resp interface{}) {
	if err, ok := resp.(error); ok {
		p.data = nil
		p.gauges = nil
		p.err = err
		return
	}

	p.data = resp.(*publisherResponse)

	// Generate pending payload gauge
	if p.data.Status.MaxPendingPayloads == 0 {
		// Support older versions where this wasn't in the response
		p.data.Status.MaxPendingPayloads = 10
	}
	p.mainGauge.Percent = int(p.data.Status.PendingPayloads * 100 / p.data.Status.MaxPendingPayloads)
	p.mainGauge.Border = false
	p.mainGauge.Label = fmt.Sprintf("%d/%d", p.data.Status.PendingPayloads, p.data.Status.MaxPendingPayloads)
}

func (p *Publisher) Draw(buf *ui.Buffer) {
	p.view.Draw(buf)
	if p.err != nil {
		return
	}

	x := p.Inner.Min.X
	y := p.Inner.Min.Y

	if p.data != nil {
		x += 1
		cells := ui.TrimCells(ui.ParseStyles(fmt.Sprintf("[Speed:](mod:bold) %.3f eps", p.data.Status.Speed), ui.Theme.Paragraph.Text), p.Inner.Dx()/2-2)
		for _, cell := range cells {
			buf.SetCell(cell, image.Point{x, y})
			x += 1
		}
		x = p.Inner.Min.X + p.Inner.Dx()/2 + 1
		cells = ui.TrimCells(ui.ParseStyles("[Pressure:](mod:bold)", ui.Theme.Paragraph.Text), p.Inner.Dx()/2-2)
		for _, cell := range cells {
			buf.SetCell(cell, image.Point{x, y})
			x += 1
		}
		p.mainGauge.SetRect(x+1, y, p.Inner.Max.X, y+1)
		p.mainGauge.Draw(buf)
		x = p.Inner.Min.X
	}

	y += 1
	buf.Fill(ui.NewCell(ui.HORIZONTAL_LINE, p.Block.BorderStyle), image.Rect(x, y, x+p.Inner.Dx(), y+1))
	buf.SetCell(ui.NewCell(ui.VERTICAL_RIGHT, p.Block.BorderStyle), image.Point{p.Min.X, y})
	buf.SetCell(ui.NewCell(ui.VERTICAL_LEFT, p.Block.BorderStyle), image.Point{p.Max.X - 1, y})
	y += 1

	// 6*3+2 for dividers and padding
	// 10 for status
	// 20 for latency
	// 20 for lines
	// 10 for pending
	// 21 for last error date (19 char RFC)
	// divide rest unevenly
	calculatedWidth := int((p.Inner.Dx() - 20 - 10 - 20 - 20 - 10 - 21) / 2)
	columnWidths := []int{int(float64(calculatedWidth) * 0.5), 10, 20, 20, 10, 21, int(float64(calculatedWidth) * 1.5)}

	var rows [][]interface{}
	if p.data == nil {
		rows = make([][]interface{}, 3)
	} else {
		rows = make([][]interface{}, 2+len(p.data.Endpoints))
	}

	rows[0] = []interface{}{"[Server](mod:bold)", "[Status](mod:bold)", "[Time(ms) Per Event](mod:bold)", "[Lines](mod:bold)", "[Pending](mod:bold)", "[Last Error Time](mod:bold)", "[Last Error](mod:bold)"}
	rows[1] = nil

	if p.data == nil {
		rows[2] = []interface{}{"Loading...", "", "", "", "", "", ""}
	} else {
		idx := 2
		for _, data := range p.data.Endpoints {
			rows[idx] = make([]interface{}, 7)
			rows[idx][0] = data.Server
			rows[idx][1] = data.Status
			rows[idx][2] = fmt.Sprintf("%.3f", data.AverageLatency)
			rows[idx][3] = fmt.Sprintf("%d", data.PublishedLines)
			rows[idx][4] = fmt.Sprintf("%d", data.PendingPayloads)
			if data.LastError == nil || data.LastErrorTime == nil {
				rows[idx][5] = ""
				rows[idx][6] = ""
			} else {
				rows[idx][5] = *data.LastErrorTime
				rows[idx][6] = *data.LastError
			}

			idx += 1
		}

		sort.Slice(rows[2:], func(i, j int) bool {
			return strings.Compare(rows[2+i][0].(string), rows[2+j][0].(string)) == -1
		})
	}

	p.table.ColumnWidths = columnWidths
	p.table.Rows = rows
	p.table.SetRect(x-1, y-1, p.Inner.Max.X+1, p.Inner.Max.Y+1)
	p.table.Draw(buf)
}
