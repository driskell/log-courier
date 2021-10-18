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
	"fmt"
	"image"

	ui "github.com/gizak/termui/v3"
)

// Table renders rows and each row has columns which can be strings or Blocks
type Table struct {
	ui.Block
	ColumnNames  []string
	ColumnWidths []int
	Rows         [][]interface{}
	firstRow     int
}

// NewTable creates a new scrollable table that can render other Blocks within cells
func NewTable() *Table {
	return &Table{
		Block: *ui.NewBlock(),
	}
}

// ScrollUp moves the viewable area upwards one row
func (t *Table) ScrollUp() {
	if t.firstRow > 0 {
		t.firstRow--
	}
}

// PageUp moves the viewable area upwards one page
func (t *Table) PageUp() {
	if t.firstRow > t.Inner.Dy()-3 {
		t.firstRow -= t.Inner.Dy() - 3
	} else {
		t.firstRow = 0
	}
}

// ScrollDown moves the viewable area downwards one row
func (t *Table) ScrollDown() {
	if t.firstRow < len(t.Rows)-(t.Inner.Dy()-3) {
		t.firstRow++
	}
}

// PageDown moves the viewable area downwards one page
func (t *Table) PageDown() {
	if t.firstRow < len(t.Rows)-(t.Inner.Dy()*2-3) {
		t.firstRow += t.Inner.Dy() - 3
	} else {
		t.firstRow = len(t.Rows) - (t.Inner.Dy() - 3)
	}
}

// SetRect implements the Drawable interface.
func (t *Table) SetRect(x1, y1, x2, y2 int) {
	t.Block.SetRect(x1, y1, x2, y2)
	if len(t.Rows)-t.Inner.Dy()-2 < 0 {
		t.firstRow = 0
	} else if t.firstRow > len(t.Rows)-t.Inner.Dy()-2 {
		t.firstRow = len(t.Rows) - t.Inner.Dy() - 2
	}
}

// Draw implements the Drawable interface
func (t *Table) Draw(buf *ui.Buffer) {
	if t.Rows == nil || len(t.Rows) == 0 {
		return
	}

	drawRow := func(row []interface{}, x int, y int) {
		for idx, columnWidth := range t.ColumnWidths {
			if drawable, ok := row[idx].(ui.Drawable); ok {
				drawable.SetRect(x-1, y, x+columnWidth+2, y+1)
				drawable.Draw(buf)
				x += columnWidth + 2
			} else {
				x++
				cells := ui.TrimCells(ui.ParseStyles(row[idx].(string), ui.Theme.Paragraph.Text), columnWidth)
				for _, cell := range cells {
					buf.SetCell(cell, image.Point{x, y})
					x++
				}
				x += columnWidth - len(cells) + 1
			}
			if idx < len(t.ColumnWidths)-1 {
				buf.SetCell(ui.NewCell(ui.VERTICAL_LINE, t.Block.BorderStyle), image.Point{x, y})
				x++
			}
		}
	}

	hasScrollbar := len(t.Rows) > t.Inner.Dy()-2

	x := t.Inner.Min.X
	y := t.Inner.Min.Y

	// If we have a border on our block, change some top cells to link to the column separator cleanly
	if t.Border {
		y = t.Min.Y
		for _, columnWidth := range t.ColumnWidths[:len(t.ColumnWidths)-1] {
			x += columnWidth + 2
			buf.SetCell(ui.NewCell(ui.HORIZONTAL_DOWN, t.Block.BorderStyle), image.Point{x, y})
			x++
		}
		x = t.Inner.Min.X
		y++
	}

	columnData := make([]interface{}, len(t.ColumnNames))
	for idx, column := range t.ColumnNames {
		columnData[idx] = fmt.Sprintf("[%s](mod:bold)", column)
	}
	drawRow(columnData, x, y)
	y++

	// Line under column names
	hvCross := ui.NewCell(HV_CROSS, t.Block.BorderStyle)
	buf.Fill(ui.NewCell(ui.HORIZONTAL_LINE, t.Block.BorderStyle), image.Rect(x, y, x+t.Inner.Dx(), y+1))
	buf.SetCell(ui.NewCell(ui.VERTICAL_RIGHT, t.Block.BorderStyle), image.Point{t.Min.X, y})
	buf.SetCell(ui.NewCell(ui.VERTICAL_LEFT, t.Block.BorderStyle), image.Point{t.Max.X - 1, y})
	for _, columnWidth := range t.ColumnWidths[:len(t.ColumnWidths)-1] {
		x += columnWidth + 2
		buf.SetCell(hvCross, image.Point{x, y})
		x++
	}
	y++
	x = t.Inner.Min.X

	// Rows
	lastRow := t.firstRow + t.Inner.Dy() - 2
	if hasScrollbar {
		lastRow--
	}
	if lastRow > len(t.Rows) {
		lastRow = len(t.Rows)
	}
	for _, row := range t.Rows[t.firstRow:lastRow] {
		drawRow(row, x, y)
		y++
	}
	// Draw the column separator into the remaining empty rows
	for ; y < t.Inner.Max.Y; y++ {
		for _, columnWidth := range t.ColumnWidths[:len(t.ColumnWidths)-1] {
			x += columnWidth + 2
			buf.SetCell(ui.NewCell(ui.VERTICAL_LINE, t.Block.BorderStyle), image.Point{x, y})
			x++
		}
		x = t.Inner.Min.X
	}
	// If we have a border on our block, change some bottom cells to link to the column separator cleanly
	if t.Border {
		y = t.Max.Y - 1
		for _, columnWidth := range t.ColumnWidths[:len(t.ColumnWidths)-1] {
			x += columnWidth + 2
			buf.SetCell(ui.NewCell(ui.HORIZONTAL_UP, t.Block.BorderStyle), image.Point{x, y})
			x++
		}
	}
	// Draw scrollbar
	if len(t.Rows) > t.Inner.Dy()-2 {
		for idx := 2; idx < t.Inner.Max.Y-t.Inner.Min.Y; idx++ {
			buf.SetCell(ui.NewCell(HEAVY_VERTICAL_LINE, t.Block.BorderStyle), image.Point{t.Max.X - 1, t.Inner.Min.Y + idx})
		}
		instructions := ui.ParseStyles(fmt.Sprintf("[Down: [d/PgDn/%c] Up: [u/PgUp/%c]](fg:black,bg:white)", ui.DOWN_ARROW, ui.UP_ARROW), ui.Theme.Paragraph.Text)
		for idx, cell := range instructions {
			buf.SetCell(cell, image.Point{t.Inner.Min.X + 1 + idx, t.Inner.Max.Y - 1})
		}
	}
	if t.firstRow != 0 {
		buf.SetCell(ui.NewCell(ui.UP_ARROW, t.Block.BorderStyle), image.Point{t.Max.X - 1, t.Inner.Min.Y + 2})
	}
	if len(t.Rows)-t.firstRow > t.Inner.Dy()-2 {
		buf.SetCell(ui.NewCell(ui.DOWN_ARROW, t.Block.BorderStyle), image.Point{t.Max.X - 1, t.Inner.Max.Y - 1})
	}
}
