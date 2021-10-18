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
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// View represents an lc-admin screen
type View interface {
	ui.Drawable
	StartUpdate()
	CompleteUpdate(interface{})
}

type view struct {
	*ui.Block
	err error
}

func newView() *view {
	return &view{
		Block: ui.NewBlock(),
	}
}

// Draw implements the Drawable interface
func (v *view) Draw(buf *ui.Buffer) {
	v.Block.Draw(buf)

	if v.err != nil {
		text := widgets.NewParagraph()
		text.Border = false
		text.Text = v.err.Error()
		text.SetRect(v.Inner.Min.X, v.Inner.Min.Y, v.Inner.Max.X, v.Inner.Max.Y)
		text.Draw(buf)
	}
}
