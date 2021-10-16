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

type Publisher struct {
	ui.Block
	client       *admin.Client
	updateChan   chan<- interface{}
	statusData   map[string]interface{}
	endpointData []interface{}
	gauges       []*widgets.Gauge
	mainGauge    *widgets.Gauge
}

func NewPublisher(client *admin.Client, updateChan chan<- interface{}) *Publisher {
	p := &Publisher{
		client:     client,
		updateChan: updateChan,
	}

	p.Block = *ui.NewBlock()
	p.mainGauge = widgets.NewGauge()

	return p
}

// StartUpdate begins a background update, and returns result on the update channel
func (p *Publisher) StartUpdate() {
	resp, err := p.client.RequestJSON("publisher")
	if err != nil {
		p.updateChan <- err
		return
	}

	p.updateChan <- resp
}

// CompleteUpdate completes the update, after which a render will occur
func (p *Publisher) CompleteUpdate(resp interface{}) {
	if resp == nil {
		p.statusData = nil
		p.endpointData = nil
		p.gauges = nil
		p.mainGauge = nil
		return
	}

	if _, ok := resp.(error); ok {
		p.statusData = nil
		p.endpointData = nil
		p.gauges = nil
		p.mainGauge = nil
		return
	}

	rootData := resp.(map[string]interface{})
	p.statusData = rootData["status"].(map[string]interface{})
	p.endpointData = rootData["endpoints"].([]interface{})

	// Generate gauges
	p.mainGauge.Percent = int(p.statusData["pendingPayloads"].(float64) * 100 / 10)
	p.mainGauge.Border = false
	if len(p.gauges) != len(p.endpointData) {
		oldGauges := p.gauges
		p.gauges = make([]*widgets.Gauge, len(p.endpointData))
		copy(p.gauges, oldGauges)
	}
	for idx, data := range p.endpointData {
		dataMap := data.(map[string]interface{})
		if p.gauges[idx] == nil {
			p.gauges[idx] = widgets.NewGauge()
		}
		// TODO: Add a maxPendingPayloads to the response
		p.gauges[idx].Percent = int(dataMap["pendingPayloads"].(float64) * 100 / 10)
		p.gauges[idx].Border = false
	}
}

func (p *Publisher) Draw(buf *ui.Buffer) {
	p.Block.Draw(buf)

	x := p.Inner.Min.X
	y := p.Inner.Min.Y

	hLine := ui.NewCell(ui.HORIZONTAL_LINE, p.Block.BorderStyle)
	vLine := ui.NewCell(ui.VERTICAL_LINE, p.Block.BorderStyle)
	hDown := ui.NewCell(ui.HORIZONTAL_DOWN, p.Block.BorderStyle)

	if p.statusData != nil {
		x += 1
		cells := ui.TrimCells(ui.ParseStyles(fmt.Sprintf("[Speed:](mod:bold) %.3f eps", p.statusData["speed"].(float64)), ui.Theme.Paragraph.Text), p.Inner.Dx()/2-2)
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
	buf.Fill(hLine, image.Rect(x, y, x+p.Inner.Dx(), y+1))
	buf.SetCell(ui.NewCell(ui.VERTICAL_RIGHT, p.Block.BorderStyle), image.Point{p.Min.X, y})
	buf.SetCell(ui.NewCell(ui.VERTICAL_LEFT, p.Block.BorderStyle), image.Point{p.Max.X - 1, y})
	y += 1

	// 4*3+2 for dividers and padding
	// 10 for status
	// 20 for latency
	// 20 for lines
	// divide amongst remaining 2 columns
	calculatedWidth := int((p.Inner.Dx() - 14 - 10 - 20 - 20) / 2)
	columnWidths := []int{calculatedWidth, 10, 20, 20, calculatedWidth}

	var rows [][]interface{}
	if p.endpointData == nil {
		rows = make([][]interface{}, 3)
	} else {
		rows = make([][]interface{}, 2+len(p.endpointData))
	}

	rows[0] = []interface{}{"[Server](mod:bold)", "[Status](mod:bold)", "[Time(ms) Per Event](mod:bold)", "[Lines](mod:bold)", "[Pressure](mod:bold)"}
	rows[1] = nil

	if p.endpointData == nil {
		rows[2] = []interface{}{"Loading...", "", "", "", ""}
	} else {
		idx := 2
		for dataIdx, data := range p.endpointData {
			dataMap := data.(map[string]interface{})
			rows[idx] = make([]interface{}, 5)
			rows[idx][0] = dataMap["server"]
			rows[idx][1] = dataMap["status"]
			rows[idx][2] = fmt.Sprintf("%.3f", dataMap["averageLatency"].(float64))
			rows[idx][3] = fmt.Sprintf("%.0f", dataMap["publishedLines"].(float64))
			rows[idx][4] = p.gauges[dataIdx]

			idx += 1
		}

		sort.Slice(rows[2:], func(i, j int) bool {
			return strings.Compare(rows[2+i][0].(string), rows[2+j][0].(string)) == -1
		})
	}

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
