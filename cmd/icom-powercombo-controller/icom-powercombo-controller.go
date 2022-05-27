package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/bbathe/icom-powercombo-controller/ui"
)

func main() {
	// show file & location, date & time
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// location for log file is in the working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("%+v", err)
		return
	}
	basefn := filepath.Join(wd, "icom-powercombo-controller")

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
