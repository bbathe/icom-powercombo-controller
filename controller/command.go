package controller

import (
	"log"

	"github.com/bbathe/icom-powercombo-controller/config"
	"github.com/bbathe/icom-powercombo-controller/data"
	"github.com/bbathe/icom-powercombo-controller/device/elecraft"
	"github.com/bbathe/icom-powercombo-controller/device/icom"
)

type command struct {
	r   *icom.Radio
	kpa *elecraft.KPA500
	kat *elecraft.KAT500
}

func (c *command) close() {
	c.r.Close()
	c.kpa.Close()
	c.kat.Close()
}

func newCommand() *command {
	// connect to radio
	r, err := icom.OpenRadio(config.Radio.CommandPort, config.Radio.Baud, config.Radio.Address)
	if err != nil {
		log.Printf("%+v", err)
		return nil
	}

	// connect to kat500
	kat, err := elecraft.OpenKAT500(config.KAT500.Port, config.KAT500.Baud)
	if err != nil {
		log.Printf("%+v", err)
		return nil
	}

	// connect to kpa500
	kpa, err := elecraft.OpenKPA500(config.KPA500.Port, config.KPA500.Baud)
	if err != nil {
		log.Printf("%+v", err)
		return nil
	}

	c := new(command)
	c.r = r
	c.kpa = kpa
	c.kat = kat

	return c
}

func (c *command) updateKPA500Mode() error {
	kpa := data.GetKPA500Data()

	// set mode on kpa500
	err := c.kpa.SetMode(kpa.Mode)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}

func (c *command) updateRadioRFPower() error {
	r := data.GetRadioData()
	kpa := data.GetKPA500Data()

	var err error
	if kpa.Mode == 1 {
		err = c.r.SetRFPower(config.Bands[r.Band].RadioRFPower.Operate)
	} else {
		err = c.r.SetRFPower(config.Bands[r.Band].RadioRFPower.Standby)
	}
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}

func (c *command) updateKPA500Band() error {
	r := data.GetRadioData()

	err := c.kpa.SetBand(r.Band)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}

func (c *command) updateKAT500Frequency() error {
	r := data.GetRadioData()

	err := c.kat.SetFrequency(r.Frequency)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}

func (c *command) setKPA500Mode(mode int) error {
	var err error

	// get current mode to tell what order to update kpa500 & radio
	kpa := data.GetKPA500Data()

	// update state
	data.KPA500{
		Mode:    mode,
		Power:   -1,
		PAVolts: -1,
		PAAmps:  -1,
	}.Update()

	if kpa.Mode > mode {
		// operate -> standby

		// update kpa500 mode
		err = c.updateKPA500Mode()
		if err != nil {
			log.Printf("%+v", err)
			return err
		}

		// update radio rf power
		err = c.updateRadioRFPower()
		if err != nil {
			log.Printf("%+v", err)
			return err
		}
	} else {
		// standby -> operate

		// update radio rf power
		err = c.updateRadioRFPower()
		if err != nil {
			log.Printf("%+v", err)
			return err
		}

		// update kpa500 mode
		err = c.updateKPA500Mode()
		if err != nil {
			log.Printf("%+v", err)
			return err
		}
	}

	return nil
}

func (c *command) getKAT500InFault() (bool, error) {
	fault, err := c.kat.GetFault()
	if err != nil {
		log.Printf("%+v", err)
		return false, err
	}

	return (fault != 0), nil
}

func (c *command) getKAT500VSWR() (float64, error) {
	vswr, err := c.kat.GetVSWR()
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	return vswr, nil
}

func (c *command) KAT500FullTune() error {
	err := c.kat.FullTune()
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}

func (c *command) getKPA500InFault() (bool, error) {
	fault, err := c.kpa.GetFault()
	if err != nil {
		log.Printf("%+v", err)
		return false, err
	}

	return (fault != 0), nil
}

func (c *command) getKPA500Power() (int, error) {
	power, err := c.kpa.GetPower()
	if err != nil {
		log.Printf("%+v", err)
		return 0, err
	}

	return power, nil
}

func (c *command) getKPA500PAVoltsCurrent() (float64, float64, error) {
	volts, amps, err := c.kpa.GetPAVoltsCurrent()
	if err != nil {
		log.Printf("%+v", err)
		return 0.0, 0.0, err
	}

	return volts, amps, nil
}
