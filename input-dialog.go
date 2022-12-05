package main

import "github.com/pwiecz/go-fltk"

type InputDialog struct {
	Win     *fltk.Window
	Input   *fltk.Input
	Message string
	Title   string
}

func NewInputDialog(title, message string) InputDialog {
	win := fltk.NewWindow(400, 200)
	win.SetLabel(title)
	b := fltk.NewBox(fltk.FLAT_BOX, 0, 0, 400, 60, message)
	b.SetLabel(message)
	i := fltk.NewInput(100, 100, 250, 30, "Enter value: ")
	i.SetTooltip("type value")
	bt_OK := fltk.NewReturnButton(100, 140, 60, 30, "OK")
	bt_OK.SetTooltip("OK")
	bt_Cancel := fltk.NewButton(180, 140, 60, 30, "Cancel")
	bt_Cancel.SetTooltip("Cancel")
	win.End()
	o := InputDialog{
		Win:   win,
		Input: i,
	}
	return o
}
