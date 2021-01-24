package ui

import (
	"log"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
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
