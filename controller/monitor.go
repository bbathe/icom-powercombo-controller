package controller

import (
	"log"
	"time"

	"github.com/bbathe/icom-powercombo-controller/config"
	"github.com/bbathe/icom-powercombo-controller/data"
	"github.com/bbathe/icom-powercombo-controller/device/icom"
	"github.com/bbathe/icom-powercombo-controller/status"
	"github.com/bbathe/icom-powercombo-controller/util"
)

type monitor struct {
	r *icom.Radio

	quit    chan bool
	qKAT500 chan bool
	qKPA500 chan bool

	freq int64
	band int

	trackKAT500 bool
}

func (m *monitor) close() {
	close(m.quit)
	close(m.qKAT500)
	close(m.qKPA500)
	m.r.Close()
}

// newMonitor spins off all the seperate processes for monitoring all devices
func newMonitor() *monitor {
	// connect to radio
	r, err := icom.OpenRadio(config.Radio.MonitorPort, config.Radio.Baud, config.Radio.Address)
	if err != nil {
		log.Printf("%+v", err)
		status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
		return nil
	}
	status.SetStatus(status.SystemStatusRadio, status.StatusOK)

	m := new(monitor)
	m.r = r
	m.trackKAT500 = true

	err = m.initializeDevices()
	if err != nil {
		log.Printf("%+v", err)
		return nil
	}

	// KAT500 monitor task
	m.qKAT500 = util.ScheduleRecurring(func() {
		// see if kat500 in fault
		f, err := controller.c.getKAT500InFault()
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)
			return
		}

		if f {
			status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)

			// don't do anything else if fault
			return
		}

		// get current vswr
		v, err := controller.c.getKAT500VSWR()
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)
			return
		}

		// update state with what we know
		data.KAT500{
			VSWR: v,
		}.Update()

		status.SetStatus(status.SystemStatusKAT500, status.StatusOK)
	}, 1*time.Second)

	// KPA500 monitor task
	m.qKPA500 = util.ScheduleRecurring(func() {
		// see if kpa500 in fault
		f, err := controller.c.getKPA500InFault()
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)
			return
		}

		if f {
			status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)

			// don't do anything else if fault
			return
		}

		// get current power level
		p, err := controller.c.getKPA500Power()
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)
			return
		}

		// get pa volts & amps
		v, a, err := controller.c.getKPA500PAVoltsCurrent()
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)
			return
		}

		// update state with what we know
		data.KPA500{
			Mode:    -1,
			Power:   p,
			PAVolts: v,
			PAAmps:  a,
		}.Update()

		status.SetStatus(status.SystemStatusKPA500, status.StatusOK)
	}, 1*time.Second)

	// kick off monitor loop
	m.quit = make(chan bool)
	go m.monitorRadio()

	return m
}

// monitorRadio keeps the KAT500 & KPA500 in-sync with the frequency on the radio
func (m *monitor) monitorRadio() {
	// while not quit
	for {
		select {
		case <-m.quit:
			return
		default:
			f, err := m.r.GetFrequency()
			if err != nil {
				log.Printf("%+v", err)
				status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
				return
			}
			status.SetStatus(status.SystemStatusRadio, status.StatusOK)

			// no update or no frequency change?
			if f < 0 || f == m.freq {
				continue
			}

			m.freq = f

			b, err := util.BandFromFrequency(f)
			if err != nil {
				continue
			}

			// update state
			data.Radio{
				Frequency: f,
				Band:      b,
			}.Update()

			//
			// coordinated frequency change across all devices
			//

			// update kat500 frequency
			if m.trackKAT500 {
				err = controller.c.updateKAT500Frequency()
				if err != nil {
					log.Printf("%+v", err)
					status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)
					continue
				}
			}

			// band change?
			if b != m.band {
				m.band = b

				// update kpa500 band
				err = controller.c.updateKPA500Band()
				if err != nil {
					log.Printf("%+v", err)
					status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)
					continue
				}

				// update radio rf power
				err = controller.c.updateRadioRFPower()
				if err != nil {
					log.Printf("%+v", err)
					status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
					continue
				}
			}
		}
	}
}

// initializeDevices makes sure the internal state is consistent with external devices
func (m *monitor) initializeDevices() error {
	var err error

	// set all device statuses based on error from any
	defer func() {
		if err != nil {
			status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
			status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)
			status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)
		} else {
			status.SetStatus(status.SystemStatusRadio, status.StatusOK)
			status.SetStatus(status.SystemStatusKAT500, status.StatusOK)
			status.SetStatus(status.SystemStatusKPA500, status.StatusOK)
		}
	}()

	var f int64
	for {
		f, err = m.r.GetFrequency()
		if err != nil {
			log.Printf("%+v", err)
			return err
		}
		if f > -1 {
			break
		}
	}

	var b int
	b, err = util.BandFromFrequency(f)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	// update our state
	m.freq = f
	m.band = b

	// update shared state
	data.Radio{
		Frequency: m.freq,
		Band:      m.band,
	}.Update()

	//
	// now get the other devices to match our internal state
	//

	// set kat500 frequency
	err = controller.c.updateKAT500Frequency()
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	// set kpa500 mode
	err = controller.c.updateKPA500Mode()
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	// set kpa500 band
	err = controller.c.updateKPA500Band()
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	// set radio rf power
	err = controller.c.updateRadioRFPower()
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}
