package controller

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

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

	freq       int64
	band       int
	lastFreqAt time.Time

	trackKAT500 atomic.Bool
}

// How long without a frequency reading before the radio LED goes red.
// The CI-V port usually stays open when the radio is powered off, so silence
// (not a serial error) is what we have to detect.
const radioSilenceLimit = 3 * time.Second

func (m *monitor) close() {
	if m == nil {
		return
	}
	if m.quit != nil {
		close(m.quit)
	}
	if m.qKAT500 != nil {
		close(m.qKAT500)
	}
	if m.qKPA500 != nil {
		close(m.qKPA500)
	}
	if m.r != nil {
		m.r.Close()
	}
}

// newMonitor spins off all the seperate processes for monitoring all devices
func newMonitor() (*monitor, error) {
	r, err := openMonitorRadio()
	if err != nil {
		log.Printf("%+v", err)
		status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
		return nil, err
	}

	ok := false
	defer func() {
		if !ok {
			_ = r.Close()
		}
	}()

	status.SetStatus(status.SystemStatusRadio, status.StatusOK)

	m := new(monitor)
	m.r = r
	m.trackKAT500.Store(true)

	err = m.initializeDevices()
	if err != nil {
		log.Printf("%+v", err)
		return nil, fmt.Errorf("device initialization failed: %w", err)
	}

	// KAT500 monitor task
	m.qKAT500 = util.ScheduleRecurring(func() {
		if controller == nil || controller.isReconnecting() {
			return
		}

		err := controller.tryPollKAT(func(cmd *command) error {
			f, err := cmd.getKAT500InFault()
			if err != nil {
				return err
			}
			if f {
				status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)
				return nil
			}

			v, err := cmd.getKAT500VSWR()
			if err != nil {
				return err
			}

			data.KAT500{VSWR: v}.Update()
			status.SetStatus(status.SystemStatusKAT500, status.StatusOK)
			return nil
		})
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)
			_ = controller.reconnect(m.quit)
		}
	}, 1*time.Second)

	// KPA500 monitor task
	m.qKPA500 = util.ScheduleRecurring(func() {
		if controller == nil || controller.isReconnecting() {
			return
		}

		err := controller.tryPollKPA(func(cmd *command) error {
			f, err := cmd.getKPA500InFault()
			if err != nil {
				return err
			}
			if f {
				status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)
				return nil
			}

			p, err := cmd.getKPA500Power()
			if err != nil {
				return err
			}

			v, a, err := cmd.getKPA500PAVoltsCurrent()
			if err != nil {
				return err
			}

			data.KPA500{
				Mode:    -1,
				Power:   p,
				PAVolts: v,
				PAAmps:  a,
			}.Update()
			status.SetStatus(status.SystemStatusKPA500, status.StatusOK)
			return nil
		})
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)
			_ = controller.reconnect(m.quit)
		}
	}, 1*time.Second)

	// kick off monitor loop
	m.quit = make(chan bool)
	go m.monitorRadio()

	ok = true
	return m, nil
}

// monitorRadio keeps the KAT500 & KPA500 in-sync with the frequency on the radio
func (m *monitor) monitorRadio() {
	for {
		select {
		case <-m.quit:
			return
		default:
			if controller != nil && controller.isReconnecting() {
				if !sleepQuit(200*time.Millisecond, m.quit) {
					return
				}
				continue
			}

			r := controller.monitorRadioPort()
			if r == nil {
				if controller == nil || !controller.reconnect(m.quit) {
					return
				}
				continue
			}

			f, err := r.GetFrequency()
			if err != nil {
				log.Printf("%+v", err)
				status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
				if !controller.reconnect(m.quit) {
					return
				}
				continue
			}

			// Empty/partial reads are normal between broadcasts.
			if f < 0 {
				if m.lastFreqAt.IsZero() || time.Since(m.lastFreqAt) <= radioSilenceLimit {
					continue
				}

				// Quiet too long — ask the radio before declaring it dead
				// (many Icoms only broadcast on VFO change).
				f, err = r.QueryFrequency()
				if err != nil {
					log.Printf("%+v", err)
					status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
					if !controller.reconnect(m.quit) {
						return
					}
					continue
				}
				if f < 0 {
					status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
					continue
				}
			}

			m.lastFreqAt = time.Now()
			status.SetStatus(status.SystemStatusRadio, status.StatusOK)

			if f == m.freq {
				continue
			}

			m.freq = f

			b, err := util.BandFromFrequency(f)
			if err != nil {
				continue
			}

			data.Radio{
				Frequency: f,
				Band:      b,
			}.Update()

			if m.trackKAT500.Load() {
				err = controller.withCommand(func(cmd *command) error {
					return cmd.updateKAT500Frequency()
				})
				if err != nil {
					log.Printf("%+v", err)
					status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)
					if !controller.reconnect(m.quit) {
						return
					}
					continue
				}
			}

			if b != m.band {
				m.band = b

				err = controller.withCommand(func(cmd *command) error {
					if err := cmd.updateKPA500Band(); err != nil {
						return err
					}
					return cmd.updateRadioRFPower()
				})
				if err != nil {
					log.Printf("%+v", err)
					status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)
					status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
					if !controller.reconnect(m.quit) {
						return
					}
					continue
				}
			}
		}
	}
}

// initializeDevices makes sure the internal state is consistent with external devices
func (m *monitor) initializeDevices() error {
	var err error

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
	for attempt := 0; attempt < 30; attempt++ {
		f, err = m.r.GetFrequency()
		if err != nil {
			log.Printf("%+v", err)
			return err
		}
		if f > -1 {
			break
		}
	}
	if f < 0 {
		err = fmt.Errorf("no frequency from radio")
		return err
	}

	var b int
	b, err = util.BandFromFrequency(f)
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	m.freq = f
	m.band = b
	m.lastFreqAt = time.Now()

	data.Radio{
		Frequency: m.freq,
		Band:      m.band,
	}.Update()

	err = controller.withCommand(func(cmd *command) error {
		if err := cmd.updateKAT500Frequency(); err != nil {
			return err
		}
		if err := cmd.updateKPA500Mode(); err != nil {
			return err
		}
		if err := cmd.updateKPA500Band(); err != nil {
			return err
		}
		return cmd.updateRadioRFPower()
	})
	if err != nil {
		log.Printf("%+v", err)
		return err
	}

	return nil
}
