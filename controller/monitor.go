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
	ctrl *Controller
	r    *icom.Radio

	quit chan bool

	freq             int64
	band             int
	lastFreqAt       time.Time
	lastRadioProbeAt time.Time
	lastKATPollAt    time.Time
	lastKPAPollAt    time.Time

	trackKAT500 atomic.Bool
}

// How long without a frequency reading before the radio LED goes red.
// The CI-V port usually stays open when the radio is powered off, so silence
// (not a serial error) is what we have to detect.
const radioSilenceLimit = 3 * time.Second

// Minimum gap between QueryFrequency liveness probes once silence is overdue.
const radioProbeInterval = 2 * time.Second

const elecraftPollInterval = 1 * time.Second

func (m *monitor) close() {
	if m == nil {
		return
	}
	if m.quit != nil {
		close(m.quit)
	}
	if m.r != nil {
		m.r.Close()
	}
}

// newMonitor spins off all the seperate processes for monitoring all devices
func newMonitor(ctrl *Controller) (*monitor, error) {
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
	m.ctrl = ctrl
	m.r = r
	m.trackKAT500.Store(true)
	now := time.Now()
	m.lastKATPollAt = now
	m.lastKPAPollAt = now

	err = m.initializeDevices()
	if err != nil {
		log.Printf("%+v", err)
		return nil, fmt.Errorf("device initialization failed: %w", err)
	}

	// Publish before the loop so monitorRadioPort can see m.r.
	m.quit = make(chan bool)
	ctrl.m = m
	go m.run()

	ok = true
	return m, nil
}

// run is the single monitor loop: CI-V listen/liveness plus due Elecraft polls.
func (m *monitor) run() {
	for {
		select {
		case <-m.quit:
			return
		default:
		}

		if m.ctrl == nil {
			return
		}
		if m.ctrl.isReconnecting() {
			if !sleepQuit(200*time.Millisecond, m.quit) {
				return
			}
			continue
		}

		if !m.stepRadio() {
			return
		}

		now := time.Now()
		if now.Sub(m.lastKATPollAt) >= elecraftPollInterval {
			m.lastKATPollAt = now
			if !m.pollKAT() {
				return
			}
		}
		if now.Sub(m.lastKPAPollAt) >= elecraftPollInterval {
			m.lastKPAPollAt = now
			if !m.pollKPA() {
				return
			}
		}
	}
}

// stepRadio performs one CI-V read/sync iteration. Returns false if the monitor should exit.
func (m *monitor) stepRadio() bool {
	r := m.ctrl.monitorRadioPort()
	if r == nil {
		return m.ctrl.reconnect(m.quit)
	}

	f, err := r.GetFrequency()
	if err != nil {
		log.Printf("%+v", err)
		status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
		return m.ctrl.reconnect(m.quit)
	}

	// Empty/partial reads are normal between broadcasts.
	if f < 0 {
		if m.lastFreqAt.IsZero() || time.Since(m.lastFreqAt) <= radioSilenceLimit {
			return true
		}

		// Quiet too long — ask the radio before declaring it dead
		// (many Icoms only broadcast on VFO change). Back off probes
		// while already failed so we do not hammer the port.
		if !m.lastRadioProbeAt.IsZero() && time.Since(m.lastRadioProbeAt) < radioProbeInterval {
			return true
		}
		m.lastRadioProbeAt = time.Now()

		f, err = r.QueryFrequency()
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
			return m.ctrl.reconnect(m.quit)
		}
		if f < 0 {
			status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
			return true
		}
	}

	m.lastFreqAt = time.Now()
	m.lastRadioProbeAt = time.Time{}
	status.SetStatus(status.SystemStatusRadio, status.StatusOK)

	if f == m.freq {
		return true
	}

	b, err := util.BandFromFrequency(f)
	if err != nil {
		// ignore out-of-band / garbage parses; do not adopt them as current freq
		return true
	}

	m.freq = f

	data.Radio{
		Frequency: f,
		Band:      b,
	}.Update()

	if m.trackKAT500.Load() {
		err = m.ctrl.withCommand(func(cmd *command) error {
			return cmd.updateKAT500Frequency()
		})
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)
			return m.ctrl.reconnect(m.quit)
		}
	}

	if b != m.band {
		m.band = b

		err = m.ctrl.withCommand(func(cmd *command) error {
			if err := cmd.updateKPA500Band(); err != nil {
				return err
			}
			return cmd.updateRadioRFPower()
		})
		if err != nil {
			log.Printf("%+v", err)
			status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)
			status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
			return m.ctrl.reconnect(m.quit)
		}
	}

	return true
}

func (m *monitor) pollKAT() bool {
	err := m.ctrl.tryPollKAT(func(cmd *command) error {
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
		return m.ctrl.reconnect(m.quit)
	}
	return true
}

func (m *monitor) pollKPA() bool {
	err := m.ctrl.tryPollKPA(func(cmd *command) error {
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
		return m.ctrl.reconnect(m.quit)
	}
	return true
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

	err = m.ctrl.withCommand(func(cmd *command) error {
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
