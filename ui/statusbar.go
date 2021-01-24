package ui

import (
	"image"
	"image/color"
	"log"

	"github.com/bbathe/icom-powercombo-controller/status"
	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
)

var (
	ivRadio  *walk.ImageView
	ivKAT500 *walk.ImageView
	ivKPA500 *walk.ImageView

	imgOK      walk.Image
	imgFailed  walk.Image
	imgUnknown walk.Image

	hStatusChangeEventHandler int
)

// statusImage returns the image to use for status based on StatusValue
func statusImage(s status.StatusValue) walk.Image {
	switch s {
	case status.StatusOK:
		return imgOK
	case status.StatusFailed:
		return imgFailed
	}

	return imgUnknown
}

// updateStatuses is the StatusChangeEventHandler
func updateStatuses(statuses []status.StatusValue) {
	if ivRadio != nil {
		err := ivRadio.SetImage(statusImage(statuses[status.SystemStatusRadio]))
		if err != nil {
			log.Printf("%+v", err)
			return
		}
	}

	if ivKAT500 != nil {
		err := ivKAT500.SetImage(statusImage(statuses[status.SystemStatusKAT500]))
		if err != nil {
			log.Printf("%+v", err)
			return
		}
	}

	if ivKPA500 != nil {
		err := ivKPA500.SetImage(statusImage(statuses[status.SystemStatusKPA500]))
		if err != nil {
			log.Printf("%+v", err)
			return
		}
	}
}

// statusBar returns a Composite that has all the controls & logic for displaying status on the main UI
func statusBar() declarative.Composite {
	var err error

	imgOK, err = walk.NewIconFromImageForDPI(generateStatusImage(color.RGBA{R: 34, G: 139, B: 34, A: 255}), 96)
	if err != nil {
		log.Printf("%+v", err)
	}

	imgFailed, err = walk.NewIconFromImageForDPI(generateStatusImage(color.RGBA{R: 237, G: 28, B: 36, A: 255}), 96)
	if err != nil {
		log.Printf("%+v", err)
	}

	imgUnknown, err = walk.NewIconFromImageForDPI(generateStatusImage(color.RGBA{R: 128, G: 128, B: 128, A: 255}), 96)
	if err != nil {
		log.Printf("%+v", err)
	}

	c := declarative.Composite{
		Layout: declarative.HBox{MarginsZero: true},
		Children: []declarative.Widget{
			declarative.ImageView{
				Image:       imgUnknown,
				AssignTo:    &ivRadio,
				ToolTipText: "Radio",
			},
			declarative.ImageView{
				Image:       imgUnknown,
				AssignTo:    &ivKAT500,
				ToolTipText: "KAT500",
			},
			declarative.ImageView{
				Image:       imgUnknown,
				AssignTo:    &ivKPA500,
				ToolTipText: "KPA500",
			},
		},
	}

	hStatusChangeEventHandler = status.Attach(updateStatuses)

	return c
}

// generateStatusImage returns an image based on the color passed, to be used for displaying the status
func generateStatusImage(clr color.Color) image.Image {
	var w, h int = 16, 8

	// create image with a filled rectangle
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, clr)
		}
	}

	return img
}
