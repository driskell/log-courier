package views

import (
	ui "github.com/gizak/termui/v3"
)

type View interface {
	ui.Drawable
	StartUpdate()
	CompleteUpdate(interface{})
}
