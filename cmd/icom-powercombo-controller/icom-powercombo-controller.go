package main

import (
	"log"
	"os"
	"path"
	"strings"

	"github.com/bbathe/icom-powercombo-controller/ui"
)

func main() {
	// show file & location, date & time
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// log file is in the same directory as the executable with the same base name
	fn, err := os.Executable()
	if err != nil {
		log.Printf("%+v", err)
		return
	}
	basefn := strings.TrimSuffix(fn, path.Ext(fn))

	// log to file
	f, err := os.OpenFile(basefn+".log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("%+v", err)
		return
	}
	defer f.Close()
	log.SetOutput(f)

	// show app, doesn't come back until main window closed
	err = ui.MainWindow()
	if err != nil {
		log.Printf("%+v", err)
		return
	}
}
