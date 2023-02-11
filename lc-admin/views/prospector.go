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

type prospectorResponse struct {
	Files []struct {
		Error     string `json:"error"`
		Id        string `json:"id"`
		Orphaned  string `json:"Orphaned"`
		Path      string `json:"path"`
		Status    string `json:"status"`
		Type      string `json:"type"`
		Harvester *struct {
			Completion     float64 `json:"completion"`
			ProcessedLines float64 `json:"processed_lines"`
		} `json:"harvester"`
	} `json:"files"`
	Status struct {
		ActiveStates int `json:"activeStates"`
		WatchedFiles int `json:"watchedFiles"`
	} `json:"status"`
}

// Prosector is a screen for monitoring open files
type Prospector struct {
	*view
	client     *admin.Client
	updateChan chan<- interface{}
	err        error
	data       *prospectorResponse
	table      *lcwidgets.Table
	gauges     []*widgets.Gauge
}

// NewProspector creates a new drawable Prospector view
func NewProspector(client *admin.Client, updateChan chan<- interface{}) View {
	p := &Prospector{
		client:     client,
		updateChan: updateChan,
	}

	p.view = newView()
	p.table = lcwidgets.NewTable()
	p.table.ColumnNames = []string{"Path", "Orphaned", "Status", "Lines", "Completion"}

	return p
}

// ScrollUp moves the viewable area upwards one row
func (p *Prospector) ScrollUp() {
	p.table.ScrollUp()
}

// PageUp moves the viewable area upwards one page
func (p *Prospector) PageUp() {
	p.table.PageUp()
}

// ScrollDown moves the viewable area downwards one row
func (p *Prospector) ScrollDown() {
	p.table.ScrollDown()
}

// PageDown moves the viewable area downwards one page
func (p *Prospector) PageDown() {
	p.table.PageDown()
}

// StartUpdate begins a background update, and returns result on the update channel
func (p *Prospector) StartUpdate() {
	var resp prospectorResponse
	if err := p.client.RequestJSON("prospector", &resp); err != nil {
		p.updateChan <- err
		return
	}

	// Sort the files - they are not sorted at the moment
	// TODO: This should be resolved in prospector side
	sort.Slice(resp.Files, func(i int, j int) bool {
		cmp := strings.Compare(resp.Files[i].Path, resp.Files[j].Path)
		if cmp == 0 {
			// Equal - compare on ID
			return strings.Compare(resp.Files[i].Id, resp.Files[j].Id) < 0
		}
		return cmp < 0
	})

	p.updateChan <- &resp
}

// CompleteUpdate completes the update, after which a render will occur
func (p *Prospector) CompleteUpdate(resp interface{}) {
	if err, ok := resp.(error); ok {
		p.data = nil
		p.gauges = nil
		p.err = err
		return
	}

	p.data = resp.(*prospectorResponse)

	// Generate gauges
	if len(p.gauges) != len(p.data.Files) {
		oldGauges := p.gauges
		p.gauges = make([]*widgets.Gauge, len(p.data.Files))
		copy(p.gauges, oldGauges)
	}
	for idx, data := range p.data.Files {
		if data.Harvester != nil {
			if p.gauges[idx] == nil {
				p.gauges[idx] = widgets.NewGauge()
			}
			p.gauges[idx].Percent = int(data.Harvester.Completion)
			p.gauges[idx].Border = false
		} else {
			p.gauges[idx] = nil
		}
	}

	var rows [][]interface{}
	if p.data == nil {
		rows = make([][]interface{}, 1)
	} else {
		rows = make([][]interface{}, len(p.data.Files))
	}

	if p.data == nil {
		rows[0] = []interface{}{"Loading...", "", "", "", ""}
	} else {
		idx := 0
		for dataIdx, data := range p.data.Files {
			rows[idx] = make([]interface{}, 5)
			rows[idx][0] = data.Path
			rows[idx][1] = data.Orphaned
			rows[idx][2] = data.Status

			if data.Harvester != nil {
				rows[idx][3] = fmt.Sprintf("%.0f", data.Harvester.ProcessedLines)
				rows[idx][4] = p.gauges[dataIdx]
			} else {
				rows[idx][3] = "-"
				rows[idx][4] = "-"
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
func (p *Prospector) SetRect(x1, y1, x2, y2 int) {
	p.view.SetRect(x1, y1, x2, y2)

	// 4*3+2 for dividers and padding
	// 10 for orphaned
	// 10 for status
	// 20 for lines
	// divide amongst remaining 2 columns
	calculatedWidth := int((p.Inner.Dx() - 14 - 10 - 10 - 20) / 2)
	columnWidths := []int{calculatedWidth, 10, 10, 20, calculatedWidth}

	p.table.ColumnWidths = columnWidths
	p.table.SetRect(p.Min.X, p.Min.Y, p.Max.X, p.Max.Y)
}

// Draw implements the Drawable interface
func (p *Prospector) Draw(buf *ui.Buffer) {
	p.view.Draw(buf)
	if p.err != nil {
		return
	}

	p.table.Draw(buf)
}
