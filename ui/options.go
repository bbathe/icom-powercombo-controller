package ui

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"

	"github.com/bbathe/icom-powercombo-controller/config"
	"github.com/lxn/walk"
	"go.bug.st/serial"

	"github.com/lxn/walk/declarative"
)

var (
	// working copy of configs
	radioConfig  config.IcomRadio
	kat500Config config.ElecraftKAT500
	kpa500Config config.ElecraftKPA500

	// band data is dynamic
	keysBands []int
	neBands   [][]*walk.NumberEdit

	// available serial ports
	ports        []string
	reNotNumbers *regexp.Regexp
)

// custom sorting for ports
type Ports []string

func (s Ports) Len() int {
	return len(s)
}

func (s Ports) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Ports) Less(i, j int) bool {
	// strip out non-numbers
	s1 := reNotNumbers.ReplaceAllString(s[i], "")
	s2 := reNotNumbers.ReplaceAllString(s[j], "")

	// get to the numbers as ints
	p1, err := strconv.Atoi(s1)
	if err != nil {
		return s[i] < s[j]
	}
	p2, err := strconv.Atoi(s2)
	if err != nil {
		return s[i] < s[j]
	}

	// return comparison
	return p1 < p2
}

// optionsWindow presents the user with the config options to change
func optionsWindow(parent *walk.MainWindow, cfn string) error {
	var (
		err       error
		configDlg *walk.Dialog
	)

	// copy config local
	radioConfig = config.Radio
	kat500Config = config.KAT500
	kpa500Config = config.KPA500

	// sort the band keys so all controls are in a logical order
	keysBands = make([]int, 0, len(config.Bands))
	for k := range config.Bands {
		keysBands = append(keysBands, k)
	}
	sort.Ints(keysBands)

	// create a place for the individual band power controls
	neBands = make([][]*walk.NumberEdit, len(config.Bands))
	for i := range keysBands {
		neBands[i] = make([]*walk.NumberEdit, 2)
	}

	// get available serial ports ready for comboboxes and sort them
	ports, err = serial.GetPortsList()
	if err != nil {
		MsgError(parent, err)
		log.Printf("%+v", err)
		return err
	}
	ports = append(ports, "")
	reNotNumbers = regexp.MustCompile(`[^\d]`)
	sort.Sort(Ports(ports))

	err = declarative.Dialog{
		AssignTo:  &configDlg,
		Title:     appName + " Options",
		Icon:      appIcon,
		FixedSize: true,
		MinSize:   declarative.Size{Width: 400},
		Font: declarative.Font{
			Family:    "MS Shell Dlg 2",
			PointSize: 10,
		},
		Layout: declarative.VBox{},
		Children: []declarative.Widget{
			declarative.TabWidget{
				Alignment: declarative.AlignHNearVNear,
				Pages: []declarative.TabPage{
					tabConfigRadio(),
					tabConfigBandPower(),
					tabConfigKAT500(),
					tabConfigKPA500(),
				},
			},
			declarative.Composite{
				Layout: declarative.HBox{},
				Children: []declarative.Widget{
					declarative.HSpacer{},
					declarative.PushButton{
						Text: "OK",
						OnClicked: func() {
							// update config
							config.Radio = radioConfig
							config.KAT500 = kat500Config
							config.KPA500 = kpa500Config

							for i, k := range keysBands {
								config.Bands[k] = config.Band{
									Low:  config.Bands[k].Low,
									High: config.Bands[k].High,
									RadioRFPower: config.RadioRFPower{
										Standby: int(neBands[i][0].Value()),
										Operate: int(neBands[i][1].Value()),
									},
								}
							}

							// persist to file
							err := config.Write(cfn)
							if err != nil {
								MsgError(parent, err)
								log.Printf("%+v", err)
							}

							configDlg.Accept()
						},
					},
					declarative.PushButton{
						Text: "Cancel",
						OnClicked: func() {
							configDlg.Cancel()
						},
					},
				},
			},
		},
	}.Create(parent)
	if err != nil {
		MsgError(parent, err)
		log.Printf("%+v", err)
		return err
	}

	// initialize control values to what's in config
	for i, k := range keysBands {
		err = neBands[i][0].SetValue(float64(config.Bands[k].RadioRFPower.Standby))
		if err != nil {
			MsgError(parent, err)
			log.Printf("%+v", err)
			return err
		}

		err = neBands[i][1].SetValue(float64(config.Bands[k].RadioRFPower.Operate))
		if err != nil {
			MsgError(parent, err)
			log.Printf("%+v", err)
			return err
		}
	}

	// start message loop
	configDlg.Run()

	return nil
}

func tabConfigRadio() declarative.TabPage {
	var cbMonitorPort *walk.ComboBox
	var cbCommandPort *walk.ComboBox
	var neBaud *walk.NumberEdit
	var leAddress *walk.LineEdit

	// find current ports
	var nMonitorPort int
	var nCommandPort int
	for n := 0; n < len(ports); n++ {
		if ports[n] == radioConfig.MonitorPort {
			nMonitorPort = n
		}
		if ports[n] == radioConfig.CommandPort {
			nCommandPort = n
		}
	}
	if nMonitorPort == len(ports) {
		nMonitorPort = 0
	}
	if nCommandPort == len(ports) {
		nCommandPort = 0
	}

	tp := declarative.TabPage{
		Title:  "Radio",
		Layout: declarative.VBox{Alignment: declarative.AlignHNearVNear},
		DataBinder: declarative.DataBinder{
			DataSource:     &radioConfig,
			ErrorPresenter: declarative.ToolTipErrorPresenter{},
		},
		Children: []declarative.Widget{
			declarative.Composite{
				Layout: declarative.VBox{},
				Children: []declarative.Widget{
					declarative.Composite{
						Layout: declarative.HBox{MarginsZero: true},
						Children: []declarative.Widget{
							declarative.HSpacer{},
							declarative.Label{
								Text:    "Monitor Port",
								MinSize: declarative.Size{Width: 100},
							},
							declarative.ComboBox{
								AssignTo:     &cbMonitorPort,
								Model:        ports,
								CurrentIndex: nMonitorPort,
								MinSize:      declarative.Size{Width: 75},
								OnCurrentIndexChanged: func() {
									radioConfig.MonitorPort = ports[cbMonitorPort.CurrentIndex()]
								},
							},
							declarative.HSpacer{},
						},
					},
					declarative.Composite{
						Layout: declarative.HBox{MarginsZero: true},
						Children: []declarative.Widget{
							declarative.HSpacer{},
							declarative.Label{
								Text:    "Command Port",
								MinSize: declarative.Size{Width: 100},
							},
							declarative.ComboBox{
								AssignTo:     &cbCommandPort,
								Model:        ports,
								CurrentIndex: nCommandPort,
								MinSize:      declarative.Size{Width: 75},
								OnCurrentIndexChanged: func() {
									radioConfig.CommandPort = ports[cbCommandPort.CurrentIndex()]
								},
							},
							declarative.HSpacer{},
						},
					},
					declarative.Composite{
						Layout: declarative.HBox{MarginsZero: true},
						Children: []declarative.Widget{
							declarative.HSpacer{},
							declarative.Label{
								Text:    "Baud",
								MinSize: declarative.Size{Width: 100},
							},
							declarative.NumberEdit{
								AssignTo: &neBaud,
								Decimals: 0,
								Value:    declarative.Bind("Baud"),
								MinSize:  declarative.Size{Width: 75},
								OnValueChanged: func() {
									radioConfig.Baud = int(neBaud.Value())
								},
							},
							declarative.HSpacer{},
						},
					},
					declarative.Composite{
						Layout: declarative.HBox{MarginsZero: true},
						Children: []declarative.Widget{
							declarative.HSpacer{},
							declarative.Label{
								Text:    "Address",
								MinSize: declarative.Size{Width: 100},
							},
							declarative.LineEdit{
								AssignTo:      &leAddress,
								Text:          declarative.Bind("Address"),
								CaseMode:      declarative.CaseModeUpper,
								TextAlignment: declarative.AlignFar,
								MaxSize:       declarative.Size{Width: 75},
								OnTextChanged: func() {
									radioConfig.Address = leAddress.Text()
								},
							},
							declarative.HSpacer{},
						},
					},
					declarative.HSpacer{},
				},
			},
		},
	}

	return tp
}

func tabConfigBandPower() declarative.TabPage {
	tp := declarative.TabPage{
		Title:    "Radio RF Power",
		Layout:   declarative.VBox{},
		Children: []declarative.Widget{},
	}

	// dynamically build band power controls
	tp.Children = make([]declarative.Widget, len(config.Bands)+1)

	// header
	tp.Children[0] = declarative.Composite{
		Font: declarative.Font{
			Family:    "MS Shell Dlg 2",
			PointSize: 10,
			Underline: true,
		},
		Layout: declarative.HBox{MarginsZero: true},
		Children: []declarative.Widget{
			declarative.HSpacer{},
			declarative.Label{
				Text:    "Band",
				MinSize: declarative.Size{Width: 40},
			},
			declarative.HSpacer{},
			declarative.Label{
				Text:    "Standby",
				MinSize: declarative.Size{Width: 90},
			},
			declarative.HSpacer{},
			declarative.Label{
				Text:    "Operate",
				MinSize: declarative.Size{Width: 70},
			},
			declarative.HSpacer{},
		},
	}

	// per band data
	for i, k := range keysBands {
		bnum := i

		tp.Children[i+1] = declarative.Composite{
			Layout: declarative.HBox{MarginsZero: true},
			Children: []declarative.Widget{
				declarative.HSpacer{},
				declarative.Label{
					Text:          fmt.Sprintf("%dm", k),
					MinSize:       declarative.Size{Width: 50},
					TextAlignment: declarative.AlignFar,
				},
				declarative.HSpacer{
					MinSize: declarative.Size{Width: 25},
					MaxSize: declarative.Size{Width: 25},
				},
				declarative.NumberEdit{
					AssignTo:           &neBands[bnum][0],
					MinSize:            declarative.Size{Width: 75},
					Decimals:           0,
					MinValue:           0,
					MaxValue:           100,
					SpinButtonsVisible: true,
					Suffix:             "%",
				},
				declarative.HSpacer{
					MinSize: declarative.Size{Width: 45},
					MaxSize: declarative.Size{Width: 45},
				},
				declarative.NumberEdit{
					AssignTo:           &neBands[bnum][1],
					MinSize:            declarative.Size{Width: 75},
					Decimals:           0,
					MinValue:           0,
					MaxValue:           100,
					SpinButtonsVisible: true,
					Suffix:             "%",
				},
				declarative.HSpacer{
					MinSize: declarative.Size{Width: 30},
					MaxSize: declarative.Size{Width: 30},
				},
			},
		}
	}

	return tp
}

func tabConfigKAT500() declarative.TabPage {
	var cbPort *walk.ComboBox
	var neBaud *walk.NumberEdit

	// find current port
	var n int
	for n = 0; n < len(ports); n++ {
		if ports[n] == kat500Config.Port {
			break
		}
	}
	if n == len(ports) {
		n = 0
	}

	return declarative.TabPage{
		Title:  "KAT500",
		Layout: declarative.VBox{Alignment: declarative.AlignHNearVNear},
		DataBinder: declarative.DataBinder{
			DataSource:     &kat500Config,
			ErrorPresenter: declarative.ToolTipErrorPresenter{},
		},
		Children: []declarative.Widget{
			declarative.Composite{
				Layout: declarative.VBox{},
				Children: []declarative.Widget{
					declarative.Composite{
						Layout: declarative.HBox{MarginsZero: true},
						Children: []declarative.Widget{
							declarative.HSpacer{},
							declarative.Label{
								Text:    "Port",
								MinSize: declarative.Size{Width: 50},
							},
							declarative.ComboBox{
								AssignTo:     &cbPort,
								Model:        ports,
								CurrentIndex: n,
								MinSize:      declarative.Size{Width: 75},
								OnCurrentIndexChanged: func() {
									kat500Config.Port = ports[cbPort.CurrentIndex()]
								},
							},
							declarative.HSpacer{},
						},
					},
					declarative.Composite{
						Layout: declarative.HBox{MarginsZero: true},
						Children: []declarative.Widget{
							declarative.HSpacer{},
							declarative.Label{
								Text:    "Baud",
								MinSize: declarative.Size{Width: 50},
							},
							declarative.NumberEdit{
								AssignTo: &neBaud,
								Decimals: 0,
								Value:    declarative.Bind("Baud"),
								MinSize:  declarative.Size{Width: 75},
								OnValueChanged: func() {
									kat500Config.Baud = int(neBaud.Value())
								},
							},
							declarative.HSpacer{},
						},
					},
				},
			},
		},
	}
}

func tabConfigKPA500() declarative.TabPage {
	var cbPort *walk.ComboBox
	var neBaud *walk.NumberEdit

	// find current port
	var n int
	for n = 0; n < len(ports); n++ {
		if ports[n] == kpa500Config.Port {
			break
		}
	}
	if n == len(ports) {
		n = 0
	}

	return declarative.TabPage{
		Title:  "KPA500",
		Layout: declarative.VBox{Alignment: declarative.AlignHNearVNear},
		DataBinder: declarative.DataBinder{
			DataSource:     &kpa500Config,
			ErrorPresenter: declarative.ToolTipErrorPresenter{},
		},
		Children: []declarative.Widget{
			declarative.Composite{
				Layout: declarative.VBox{},
				Children: []declarative.Widget{
					declarative.Composite{
						Layout: declarative.HBox{MarginsZero: true},
						Children: []declarative.Widget{
							declarative.HSpacer{},
							declarative.Label{
								Text:    "Port",
								MinSize: declarative.Size{Width: 50},
							},
							declarative.ComboBox{
								AssignTo:     &cbPort,
								Model:        ports,
								CurrentIndex: n,
								MinSize:      declarative.Size{Width: 75},
								OnCurrentIndexChanged: func() {
									kpa500Config.Port = ports[cbPort.CurrentIndex()]
								},
							},
							declarative.HSpacer{},
						},
					},
					declarative.Composite{
						Layout: declarative.HBox{MarginsZero: true},
						Children: []declarative.Widget{
							declarative.HSpacer{},
							declarative.Label{
								Text:    "Baud",
								MinSize: declarative.Size{Width: 50},
							},
							declarative.NumberEdit{
								AssignTo: &neBaud,
								Decimals: 0,
								Value:    declarative.Bind("Baud"),
								MinSize:  declarative.Size{Width: 75},
								OnValueChanged: func() {
									kpa500Config.Baud = int(neBaud.Value())
								},
							},
							declarative.HSpacer{},
						},
					},
				},
			},
		},
	}
}
