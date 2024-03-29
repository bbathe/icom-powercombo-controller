package ui

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bbathe/icom-powercombo-controller/config"
	"github.com/bbathe/icom-powercombo-controller/controller"
	"github.com/bbathe/icom-powercombo-controller/data"
	"github.com/bbathe/icom-powercombo-controller/status"
	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

var (
	mainWin *walk.MainWindow

	appName     = "icom-powercombo-controller"
	appIcon     *walk.Icon
	fontBold    *walk.Font
	fontNotBold *walk.Font

	flashWindowEx *windows.Proc

	mutexCtrl sync.Mutex
	ctrl      *controller.Controller

	hDataChangeHandler int
)

func init() {
	var err error

	// load app icon
	appIcon, err = walk.Resources.Icon("2")
	if err != nil {
		log.Printf("%+v", err)
		return
	}

	// fonts
	fontBold, err = walk.NewFont("MS Shell Dlg 2", 10, walk.FontBold)
	if err != nil {
		log.Printf("%+v", err)
		return
	}

	fontNotBold, err = walk.NewFont("MS Shell Dlg 2", 10, 0)
	if err != nil {
		log.Printf("%+v", err)
		return
	}

	// get to FlashWindowEx
	winuserDll, err := windows.LoadDLL("User32.dll")
	if err != nil {
		log.Fatalf("%+v", err)
	}

	flashWindowEx, err = winuserDll.FindProc("FlashWindowEx")
	if err != nil {
		log.Fatalf("%+v", err)
	}
}

// MainWindow finishes initialization and gets everything going
func MainWindow() error {
	var (
		err        error
		configFile string

		sKPA500Mode *walk.Slider
		tlStandby   *walk.TextLabel
		tlOperate   *walk.TextLabel
		tlWatts     *walk.TextLabel
		tlVolts     *walk.TextLabel
		tlAmps      *walk.TextLabel
		tlVSWR      *walk.TextLabel

		actTrackKAT500 *walk.Action
	)

	// our main window
	err = declarative.MainWindow{
		AssignTo: &mainWin,
		Title:    appName,
		Icon:     appIcon,
		Visible:  false,
		Font: declarative.Font{
			Family:    "MS Shell Dlg 2",
			PointSize: 10,
		},
		ContextMenuItems: []declarative.MenuItem{
			declarative.Action{
				Text: "&Options...",
				OnTriggered: func() {
					// go into standby
					sKPA500Mode.SetValue(0)

					updateConfig(mainWin, configFile)
				},
			},
			declarative.Action{
				Text: "&Initiate Full Tune",
				OnTriggered: func() {
					if ctrl != nil {
						MsgBusyWithTask(mainWin, "Tune in progress...", func() {
							err := ctrl.KAT500FullTune()
							if err != nil {
								log.Printf("%+v", err)
								return
							}
						})
					}
				},
			},
			declarative.Action{
				AssignTo: &actTrackKAT500,
				Text:     "&Track KAT500",
				Checked:  true,
				OnTriggered: func() {
					if ctrl != nil {
						// flip action controller takes on frequency changes
						if actTrackKAT500.Checked() {
							ctrl.SetTrackKAT500(false)
						} else {
							ctrl.SetTrackKAT500(true)
						}

						// flip checked indicator
						err := actTrackKAT500.SetChecked(!actTrackKAT500.Checked())
						if err != nil {
							log.Printf("%+v", err)
							return
						}
					}
				},
			},
		},
		Layout: declarative.Grid{
			Columns:   1,
			Alignment: declarative.AlignHCenterVCenter,
		},
		OnSizeChanged: func() {
			err := resetMainWinSize(mainWin)
			if err != nil {
				MsgError(nil, err)
				log.Printf("%+v", err)
			}
		},
		Children: []declarative.Widget{
			declarative.Composite{
				Layout: declarative.Grid{Columns: 1},
				Border: true,
				Children: []declarative.Widget{
					declarative.Composite{
						Layout: declarative.Grid{
							Rows:        1,
							MarginsZero: true,
						},
						Children: []declarative.Widget{
							declarative.Slider{
								AssignTo:       &sKPA500Mode,
								MinSize:        declarative.Size{Height: 48},
								MaxValue:       1,
								MinValue:       0,
								ToolTipsHidden: true,
								Orientation:    declarative.Vertical,
								OnValueChanged: func() {
									m := sKPA500Mode.Value()

									if ctrl != nil {
										err := ctrl.SetKPA500Mode(m)
										if err != nil {
											log.Printf("%+v", err)
											return
										}
									}

									if m == 1 {
										tlOperate.SetFont(fontBold)
										tlStandby.SetFont(fontNotBold)
									} else {
										tlOperate.SetFont(fontNotBold)
										tlStandby.SetFont(fontBold)
									}

								},
							},
							declarative.Composite{
								Layout: declarative.Grid{
									Columns: 1,
									Margins: declarative.Margins{
										Top:    4,
										Bottom: 5,
									},
								},
								Children: []declarative.Widget{
									declarative.TextLabel{
										Text:     "Standby",
										AssignTo: &tlStandby,
									},
									declarative.VSpacer{},
									declarative.TextLabel{
										Text:     "Operate",
										AssignTo: &tlOperate,
									},
								},
							},
						},
					},
					declarative.Composite{
						Layout: declarative.VBox{MarginsZero: true},
						Children: []declarative.Widget{
							declarative.Composite{
								Layout: declarative.HBox{MarginsZero: true},
								Children: []declarative.Widget{
									declarative.TextLabel{
										AssignTo: &tlWatts,
									},
									declarative.TextLabel{
										AssignTo: &tlVolts,
									},
									declarative.TextLabel{
										AssignTo: &tlAmps,
									},
								},
							},
							declarative.Composite{
								Layout: declarative.HBox{MarginsZero: true},
								Children: []declarative.Widget{
									declarative.HSpacer{},
									declarative.TextLabel{
										AssignTo: &tlVSWR,
									},
									declarative.HSpacer{},
								},
							},
						},
					},
				},
			},
			statusBar(),
		},
	}.Create()
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	// update controls with data from devices
	hDataChangeHandler = data.Attach(func(d data.Data) {
		err := tlWatts.SetText(fmt.Sprintf("%d w", d.KPA500.Power))
		if err != nil {
			log.Printf("%+v", err)
		}
		err = tlVolts.SetText(fmt.Sprintf("%.1f v", d.KPA500.PAVolts))
		if err != nil {
			log.Printf("%+v", err)
		}
		err = tlAmps.SetText(fmt.Sprintf("%.1f a", d.KPA500.PAAmps))
		if err != nil {
			log.Printf("%+v", err)
		}
		err = tlVSWR.SetText(fmt.Sprintf("%.2f vswr", d.KAT500.VSWR))
		if err != nil {
			log.Printf("%+v", err)
		}
	})

	// disable maximize and resizing
	hwnd := mainWin.Handle()
	win.SetWindowLong(hwnd, win.GWL_STYLE, win.GetWindowLong(hwnd, win.GWL_STYLE) & ^(win.WS_MAXIMIZEBOX|win.WS_SIZEBOX))

	// get configFile name
	configFile, err = determineConfigFile()
	if err != nil {
		MsgError(nil, err)
		log.Printf("%+v", err)
		return err
	}

	// read config
	newConfig, err := config.ReadOrCreate(configFile)
	if err != nil {
		MsgError(nil, err)
		log.Printf("%+v", err)
		return err
	}

	// set window position based on config
	err = mainWin.SetY(config.UI.MainWinPosition.Y)
	if err != nil {
		MsgError(nil, err)
		log.Printf("%+v", err)
		return err
	}
	err = mainWin.SetX(config.UI.MainWinPosition.X)
	if err != nil {
		MsgError(nil, err)
		log.Printf("%+v", err)
		return err
	}
	err = resetMainWinSize(mainWin)
	if err != nil {
		MsgError(nil, err)
		log.Printf("%+v", err)
		return err
	}

	// on window close
	mainWin.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		MsgBusyWithTask(mainWin, "Shutting down...", func() {
			// unhook subscribed handlers so we don't try to update UI elements
			data.Detach(hDataChangeHandler)
			status.Detach(hStatusChangeEventHandler)

			// save windows position in config
			config.UI.MainWinPosition.FromBounds(mainWin.Bounds())
			err = config.Write(configFile)
			if err != nil {
				MsgError(nil, err)
				log.Printf("%+v", err)
			}

			// shutdown controller
			mutexCtrl.Lock()
			defer mutexCtrl.Unlock()

			// shutdown in standby
			err = ctrl.SetKPA500Mode(0)
			if err != nil {
				MsgError(nil, err)
				log.Printf("%+v", err)
			}

			if ctrl != nil {
				ctrl.Close()
			}
		})
	})

	// make visible
	mainWin.SetVisible(true)

	// need the user to set config?
	if newConfig {
		// controller started on return from updateConfig
		updateConfig(mainWin, configFile)
	} else {
		func() {
			// start controller
			mutexCtrl.Lock()
			defer mutexCtrl.Unlock()

			ctrl = controller.NewController()
		}()
	}

	// startup in standby
	sKPA500Mode.SetValue(0)

	// start message loop, returns when window closed
	mainWin.Run()

	return nil
}

// updateConfig shuts down the device coordination and presents the user with options dialog
func updateConfig(p *walk.MainWindow, configFile string) {
	mutexCtrl.Lock()
	defer mutexCtrl.Unlock()

	MsgBusyWithTask(p, "Stopping processes...", func() {
		// stop controller while config being changed
		if ctrl != nil {
			ctrl.Close()
		}
	})

	// prompt user to make changes
	err := optionsWindow(p, configFile)
	if err != nil {
		MsgError(p, err)
		log.Printf("%+v", err)
		return
	}

	// start controller
	ctrl = controller.NewController()
}

// determineConfigFile returns the configuration file to use based on whether user passed one on the commandline
func determineConfigFile() (string, error) {
	var cfn string

	flg := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flg.StringVar(&cfn, "config", "", "")
	err := flg.Parse(os.Args[1:])
	if err != nil {
		exefn, _ := os.Executable()
		basefn := strings.TrimSuffix(filepath.Base(exefn), path.Ext(exefn))

		log.Printf("%+v", err)
		return "", fmt.Errorf("%s\n\nUsage of %s\n  -config string\n    Configuration file", err.Error(), basefn)
	}

	// if user passed a filename, use that
	if len(cfn) > 0 {
		return cfn, nil
	}

	// default config file is in the working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("%+v", err)
		return "", err
	}
	configFile := filepath.Join(wd, "icom-powercombo-controller") + ".yaml"

	return configFile, nil
}

// resetMainWinSize resets the windows height & width
func resetMainWinSize(w *walk.MainWindow) error {
	err := w.SetWidth(180)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}
	err = w.SetHeight(0)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}
