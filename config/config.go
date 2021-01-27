package config

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/lxn/walk"
	"gopkg.in/yaml.v2"
)

type mainwinposition struct {
	X int `yaml:"topleftx"`
	Y int `yaml:"toplefty"`
}

func (mwp *mainwinposition) FromBounds(bounds walk.Rectangle) {
	mwp.X = bounds.X
	mwp.Y = bounds.Y
}

type ui struct {
	MainWinPosition mainwinposition
}

type IcomRadio struct {
	MonitorPort string
	CommandPort string
	Baud        int
	Address     string
}

type ElecraftKAT500 struct {
	Port string
	Baud int
}

type ElecraftKPA500 struct {
	Port string
	Baud int
}

type RadioRFPower struct {
	Standby int
	Operate int
}

type Band struct {
	Low          int64
	High         int64
	RadioRFPower RadioRFPower
}

// Configuration is the struct that is serialized to file
type Configuration struct {
	UI     ui
	Radio  IcomRadio
	KAT500 ElecraftKAT500
	KPA500 ElecraftKPA500
	Bands  map[int]Band
}

var (
	// unwrapped config values
	UI     ui
	Radio  IcomRadio
	KAT500 ElecraftKAT500
	KPA500 ElecraftKPA500
	Bands  map[int]Band
)

// Read loads application configuration from file fname
func Read(fname string) error {
	// read yaml from fname
	bytes, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	// yaml to struct
	var c Configuration
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	// unwrap config values
	UI = c.UI
	Radio = c.Radio
	KAT500 = c.KAT500
	KPA500 = c.KPA500
	Bands = c.Bands

	return nil
}

// ReadOrCreate loads application configuration or creates a new file named fname
// returns true if config was "created"
func ReadOrCreate(fname string) (bool, error) {
	// test if file exists, if not create it
	_, err := os.Stat(fname)
	if err != nil {
		if os.IsNotExist(err) {
			// bootstrap some defaults
			UI = ui{
				MainWinPosition: mainwinposition{
					X: 200,
					Y: 200,
				},
			}

			Radio = IcomRadio{}
			KAT500 = ElecraftKAT500{}
			KPA500 = ElecraftKPA500{}

			Bands = make(map[int]Band)
			Bands[6] = Band{Low: 50000000, High: 54000000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 30}}
			Bands[10] = Band{Low: 28000000, High: 29700000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 30}}
			Bands[12] = Band{Low: 24890000, High: 24990000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 30}}
			Bands[15] = Band{Low: 21000000, High: 21450000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 30}}
			Bands[17] = Band{Low: 18068000, High: 18168000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 30}}
			Bands[20] = Band{Low: 14000000, High: 14350000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 30}}
			Bands[30] = Band{Low: 10100000, High: 10150000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 10}}
			Bands[40] = Band{Low: 7000000, High: 7300000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 30}}
			Bands[60] = Band{Low: 5240000, High: 5500000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 5}}
			Bands[80] = Band{Low: 3500000, High: 4000000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 30}}
			Bands[160] = Band{Low: 1800000, High: 2000000, RadioRFPower: RadioRFPower{Standby: 100, Operate: 30}}

			return true, nil
		}
	}

	return false, Read(fname)
}

// Write writes application configuration to the file named fname
func Write(fname string) error {
	// wrap config values
	c := Configuration{
		UI:     UI,
		Radio:  Radio,
		KAT500: KAT500,
		KPA500: KPA500,
		Bands:  Bands,
	}

	// struct to yaml
	b, err := yaml.Marshal(c)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	// write out to fname
	err = ioutil.WriteFile(fname, b, 0600)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}
