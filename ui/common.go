package ui

import (
	"log"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

// MsgError displays dialog to user with error details
func MsgError(p walk.Form, err error) {
	if p == nil {
		walk.MsgBox(nil, appName, err.Error(), walk.MsgBoxIconError|walk.MsgBoxServiceNotification)
	} else {
		walk.MsgBox(p, appName, err.Error(), walk.MsgBoxIconError)
	}
}

// MsgBusyWithTask executes fn while showing the user that we are doing something
func MsgBusyWithTask(p walk.Form, title string, fn func()) {
	var dlg *walk.Dialog

	err := declarative.Dialog{
		AssignTo:  &dlg,
		Title:     title,
		Icon:      appIcon,
		FixedSize: true,
		MinSize:   declarative.Size{Width: 250},
		Font: declarative.Font{
			Family:    "MS Shell Dlg 2",
			PointSize: 10,
		},
		Layout: declarative.VBox{},
	}.Create(p)
	if err != nil {
		MsgError(p, err)
		log.Printf("%+v", err)
		return
	}

	dlg.Starting().Attach(func() {
		// call passed func
		fn()

		// close dialog
		dlg.Accept()
	})

	dlg.Run()
}

func flashWindow(parent *walk.MainWindow, times uint32) {
	if parent == nil {
		return
	}

	type flashwinfo struct {
		CbSize    uint32
		Hwnd      win.HWND
		DwFlags   uint32
		UCount    uint32
		DwTimeout uint32
	}

	// flash both the window caption and taskbar button
	fw := flashwinfo{
		Hwnd:    parent.Handle(),
		DwFlags: 0x00000003,
		UCount:  times,
	}
	fw.CbSize = uint32(unsafe.Sizeof(fw))

	// flash continuously until the window comes to the foreground?
	if times == 0 {
		fw.DwFlags |= 0x0000000C
	}

	_, _, _ = flashWindowEx.Call(uintptr(unsafe.Pointer(&fw)))
}
