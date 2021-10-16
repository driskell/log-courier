package views

import (
	"fmt"
	"image"
	"sort"
	"strings"

	"github.com/driskell/log-courier/lc-lib/admin"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type Prospector struct {
	ui.Block
	client     *admin.Client
	updateChan chan<- interface{}
	data       []interface{}
	gauges     []*widgets.Gauge
}

func NewProspector(client *admin.Client, updateChan chan<- interface{}) *Prospector {
	p := &Prospector{
		client:     client,
		updateChan: updateChan,
	}

	p.Block = *ui.NewBlock()

	return p
}

// StartUpdate begins a background update, and returns result on the update channel
func (p *Prospector) StartUpdate() {
	resp, err := p.client.RequestJSON("prospector/files")
	if err != nil {
		p.updateChan <- err
		return
	}

	p.updateChan <- resp
}

// CompleteUpdate completes the update, after which a render will occur
func (p *Prospector) CompleteUpdate(resp interface{}) {
	if resp == nil {
		p.data = nil
		p.gauges = nil
		return
	}

	if _, ok := resp.(error); ok {
		p.data = nil
		p.gauges = nil
		return
	}

	p.data = resp.([]interface{})

	// Generate gauges
	if len(p.gauges) != len(p.data) {
		oldGauges := p.gauges
		p.gauges = make([]*widgets.Gauge, len(p.data))
		copy(p.gauges, oldGauges)
	}
	for idx, data := range p.data {
		dataMap := data.(map[string]interface{})
		if harvesterData, ok := dataMap["harvester"].(map[string]interface{}); ok {
			if p.gauges[idx] == nil {
				p.gauges[idx] = widgets.NewGauge()
			}
			p.gauges[idx].Percent = int(harvesterData["completion"].(float64))
			p.gauges[idx].Border = false
		} else {
			p.gauges[idx] = nil
		}
	}
}

func (p *Prospector) Draw(buf *ui.Buffer) {
	p.Block.Draw(buf)

	// 4*3+2 for dividers and padding
	// 10 for orphaned
	// 10 for status
	// 20 for lines
	// divide amongst remaining 2 columns
	calculatedWidth := int((p.Inner.Dx() - 14 - 10 - 10 - 20) / 2)
	columnWidths := []int{calculatedWidth, 10, 10, 20, calculatedWidth}

	var rows [][]interface{}
	if p.data == nil {
		rows = make([][]interface{}, 3)
	} else {
		rows = make([][]interface{}, 2+len(p.data))
	}

	rows[0] = []interface{}{"[Path](mod:bold)", "[Orphaned](mod:bold)", "[Status](mod:bold)", "[Lines](mod:bold)", "[Completion](mod:bold)"}
	rows[1] = nil

	if p.data == nil {
		rows[2] = []interface{}{"Loading...", "", "", "", ""}
	} else {
		idx := 2
		for dataIdx, data := range p.data {
			dataMap := data.(map[string]interface{})
			rows[idx] = make([]interface{}, 5)
			rows[idx][0] = dataMap["path"]
			rows[idx][1] = dataMap["orphaned"]
			rows[idx][2] = dataMap["status"]

			if harvesterData, ok := dataMap["harvester"].(map[string]interface{}); ok {
				rows[idx][3] = fmt.Sprintf("%.0f", harvesterData["processed_lines"].(float64))
				rows[idx][4] = p.gauges[dataIdx]
			} else {
				rows[idx][3] = "-"
				rows[idx][4] = "-"
			}

			idx += 1
		}

		sort.Slice(rows[2:], func(i, j int) bool {
			return strings.Compare(rows[2+i][0].(string), rows[2+j][0].(string)) == -1
		})
	}

	x := p.Inner.Min.X
	y := p.Inner.Min.Y
	hLine := ui.NewCell(ui.HORIZONTAL_LINE, p.Block.BorderStyle)
	vLine := ui.NewCell(ui.VERTICAL_LINE, p.Block.BorderStyle)
	hDown := ui.NewCell(ui.HORIZONTAL_DOWN, p.Block.BorderStyle)
	for _, columnWidth := range columnWidths[:len(columnWidths)-1] {
		x += columnWidth + 2
		buf.SetCell(hDown, image.Point{x, y - 1})
		x += 1
	}
	x = p.Inner.Min.X
	for _, row := range rows {
		if row == nil {
			hvCross := ui.NewCell('┼', p.Block.BorderStyle)
			buf.Fill(hLine, image.Rect(x, y, x+p.Inner.Dx(), y+1))
			buf.SetCell(ui.NewCell(ui.VERTICAL_RIGHT, p.Block.BorderStyle), image.Point{p.Min.X, y})
			buf.SetCell(ui.NewCell(ui.VERTICAL_LEFT, p.Block.BorderStyle), image.Point{p.Max.X - 1, y})
			for _, columnWidth := range columnWidths[:len(columnWidths)-1] {
				x += columnWidth + 2
				buf.SetCell(hvCross, image.Point{x, y})
				x += 1
			}
			y += 1
			x = p.Inner.Min.X
			continue
		}
		for idx, columnWidth := range columnWidths {
			if gauge, ok := row[idx].(*widgets.Gauge); ok {
				gauge.SetRect(x-1, y, x+columnWidth+2, y+1)
				gauge.Draw(buf)
				x += columnWidth + 3
				continue
			}
			x += 1
			cells := ui.TrimCells(ui.ParseStyles(row[idx].(string), ui.Theme.Paragraph.Text), columnWidth)
			for _, cell := range cells {
				buf.SetCell(cell, image.Point{x, y})
				x += 1
			}
			x += columnWidth - len(cells) + 1
			buf.SetCell(vLine, image.Point{x, y})
			x += 1
		}
		y += 1
		x = p.Inner.Min.X
	}
}
