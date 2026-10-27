/*
 * Copyright 2026 Jason Woods and contributors
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

	"github.com/driskell/log-courier/lc-admin/lcwidgets"
	"github.com/driskell/log-courier/lc-lib/admin"
	ui "github.com/gizak/termui/v3"
)

type processorResponse struct {
	Routines []struct {
		ID              int      `json:"id"`
		ProcessedEvents int64    `json:"processedEvents"`
		Speed           *float64 `json:"speed"`
	} `json:"routines"`
}

// Processor is a screen for monitoring the processor routines
type Processor struct {
	*view
	client     *admin.Client
	updateChan chan<- interface{}
	data       *processorResponse
	err        error
	table      *lcwidgets.Table
}

// NewProcessor creates a new drawable Processor view
func NewProcessor(client *admin.Client, updateChan chan<- interface{}) View {
	p := &Processor{
		client:     client,
		updateChan: updateChan,
	}

	p.view = newView()
	p.table = lcwidgets.NewTable()
	p.table.ColumnNames = []string{"Routine", "Events", "Events/s"}

	return p
}

// ScrollUp moves the viewable area upwards one row
func (p *Processor) ScrollUp() {
	p.table.ScrollUp()
}

// PageUp moves the viewable area upwards one page
func (p *Processor) PageUp() {
	p.table.PageUp()
}

// ScrollDown moves the viewable area downwards one row
func (p *Processor) ScrollDown() {
	p.table.ScrollDown()
}

// PageDown moves the viewable area downwards one page
func (p *Processor) PageDown() {
	p.table.PageDown()
}

// StartUpdate begins a background update, and returns result on the update channel
func (p *Processor) StartUpdate() {
	var resp processorResponse
	if err := p.client.RequestJSON("processor", &resp); err != nil {
		p.updateChan <- err
		return
	}

	sort.SliceStable(resp.Routines, func(i int, j int) bool {
		return resp.Routines[i].ID < resp.Routines[j].ID
	})

	p.updateChan <- &resp
}

// CompleteUpdate completes the update, after which a render will occur
func (p *Processor) CompleteUpdate(resp interface{}) {
	if err, ok := resp.(error); ok {
		p.data = nil
		p.err = err
		return
	}

	p.data = resp.(*processorResponse)

	rows := make([][]interface{}, 0, len(p.data.Routines))
	for _, data := range p.data.Routines {
		rows = append(rows, []interface{}{fmt.Sprintf("%d", data.ID), fmt.Sprintf("%d", data.ProcessedEvents), formatSpeed(data.Speed)})
	}

	p.table.Rows = rows
}

// SetRect implements the Drawable interface
func (p *Processor) SetRect(x1, y1, x2, y2 int) {
	p.view.SetRect(x1, y1, x2, y2)

	// 2*3+2 for dividers and padding, divided amongst the 3 columns
	calculatedWidth := int((p.Inner.Dx() - 8) / 3)
	p.table.ColumnWidths = []int{calculatedWidth, calculatedWidth, calculatedWidth}
	p.table.SetRect(p.Min.X, p.Min.Y, p.Max.X, p.Max.Y)
}

// Draw implements the Drawable interface
func (p *Processor) Draw(buf *ui.Buffer) {
	p.view.Draw(buf)
	if p.err != nil {
		return
	}

	p.table.Draw(buf)
}
