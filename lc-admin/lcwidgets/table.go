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

package lcwidgets

import (
	"image"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type Table struct {
	ui.Block
	ColumnWidths []int
	Rows         [][]interface{}
}

func NewTable() *Table {
	return &Table{
		Block: *ui.NewBlock(),
	}
}

func (t *Table) Draw(buf *ui.Buffer) {
	if t.Rows == nil || len(t.Rows) == 0 {
		return
	}

	x := t.Inner.Min.X
	y := t.Inner.Min.Y

	columnWidths := t.ColumnWidths
	if columnWidths == nil {
		columnWidths = make([]int, len(t.Rows[0]))
		autoWidth := t.Inner.Dx() / len(t.Rows[0])
		for idx := 0; idx < len(t.Rows[0]); idx += 1 {
			columnWidths[idx] = autoWidth
		}
	}

	if t.Border {
		y = t.Min.Y
		for _, columnWidth := range columnWidths[:len(columnWidths)-1] {
			x += columnWidth + 2
			buf.SetCell(ui.NewCell(ui.HORIZONTAL_DOWN, t.Block.BorderStyle), image.Point{x, y})
			x += 1
		}
		x = t.Inner.Min.X
		y += 1
	}
	for _, row := range t.Rows {
		if row == nil {
			hvCross := ui.NewCell('┼', t.Block.BorderStyle)
			buf.Fill(ui.NewCell(ui.HORIZONTAL_LINE, t.Block.BorderStyle), image.Rect(x, y, x+t.Inner.Dx(), y+1))
			buf.SetCell(ui.NewCell(ui.VERTICAL_RIGHT, t.Block.BorderStyle), image.Point{t.Min.X, y})
			buf.SetCell(ui.NewCell(ui.VERTICAL_LEFT, t.Block.BorderStyle), image.Point{t.Max.X - 1, y})
			for _, columnWidth := range columnWidths[:len(columnWidths)-1] {
				x += columnWidth + 2
				buf.SetCell(hvCross, image.Point{x, y})
				x += 1
			}
			y += 1
			x = t.Inner.Min.X
			continue
		}
		for idx, columnWidth := range columnWidths {
			if gauge, ok := row[idx].(*widgets.Gauge); ok {
				gauge.SetRect(x-1, y, x+columnWidth+2, y+1)
				gauge.Draw(buf)
				x += columnWidth + 2
			} else {
				x += 1
				cells := ui.TrimCells(ui.ParseStyles(row[idx].(string), ui.Theme.Paragraph.Text), columnWidth)
				for _, cell := range cells {
					buf.SetCell(cell, image.Point{x, y})
					x += 1
				}
				x += columnWidth - len(cells) + 1
			}
			if idx < len(columnWidths)-1 {
				buf.SetCell(ui.NewCell(ui.VERTICAL_LINE, t.Block.BorderStyle), image.Point{x, y})
				x += 1
			}
		}
		y += 1
		x = t.Inner.Min.X
	}
	for ; y < t.Inner.Max.Y; y += 1 {
		for _, columnWidth := range columnWidths[:len(columnWidths)-1] {
			x += columnWidth + 2
			buf.SetCell(ui.NewCell(ui.VERTICAL_LINE, t.Block.BorderStyle), image.Point{x, y})
			x += 1
		}
		x = t.Inner.Min.X
	}
	if t.Border {
		y = t.Max.Y - 1
		for _, columnWidth := range columnWidths[:len(columnWidths)-1] {
			x += columnWidth + 2
			buf.SetCell(ui.NewCell(ui.HORIZONTAL_UP, t.Block.BorderStyle), image.Point{x, y})
			x += 1
		}
	}
}
