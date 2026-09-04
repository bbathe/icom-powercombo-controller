package controller

import (
	"fmt"
	"log"
	"time"

	"github.com/bbathe/icom-powercombo-controller/config"
	"github.com/bbathe/icom-powercombo-controller/device/icom"
	"github.com/bbathe/icom-powercombo-controller/status"
)

const (
	reconnectBackoffInitial = 1 * time.Second
	reconnectBackoffMax     = 30 * time.Second
)

// reconnect restores all serial ports after a communication failure.
// Concurrent callers wait for an in-progress attempt. Returns false if quit fires.
func (c *Controller) reconnect(quit <-chan bool) bool {
	if c == nil {
		return false
	}

	// single-flight: wait for an in-progress reconnect to finish
	for {
		if c.reconnectMu.TryLock() {
			break
		}
		select {
		case <-quit:
			return false
		case <-time.After(200 * time.Millisecond):
			if !c.reconnecting.Load() {
				return true
			}
		}
	}
	defer c.reconnectMu.Unlock()

	c.reconnecting.Store(true)
	defer c.reconnecting.Store(false)

	backoff := reconnectBackoffInitial

	for {
		select {
		case <-quit:
			return false
		default:
		}

		status.SetStatus(status.SystemStatusRadio, status.StatusFailed)
		status.SetStatus(status.SystemStatusKAT500, status.StatusFailed)
		status.SetStatus(status.SystemStatusKPA500, status.StatusFailed)

		log.Printf("reconnecting serial devices (backoff %s)", backoff)

		if err := c.reopenAllPorts(); err != nil {
			log.Printf("%+v", err)
			if !sleepQuit(backoff, quit) {
				return false
			}
			backoff = nextBackoff(backoff)
			continue
		}

		if err := c.m.initializeDevices(); err != nil {
			log.Printf("%+v", err)
			if !sleepQuit(backoff, quit) {
				return false
			}
			backoff = nextBackoff(backoff)
			continue
		}

		log.Printf("reconnect succeeded")
		return true
	}
}

func (c *Controller) reopenAllPorts() error {
	c.portsMu.Lock()
	oldCmd := c.c
	var oldMon *icom.Radio
	if c.m != nil {
		oldMon = c.m.r
		c.m.r = nil
	}
	c.c = nil
	c.portsMu.Unlock()

	if oldCmd != nil {
		oldCmd.close()
	}
	if oldMon != nil {
		_ = oldMon.Close()
	}

	newCmd, err := newCommand()
	if err != nil {
		return err
	}

	newMon, err := openMonitorRadio()
	if err != nil {
		newCmd.close()
		return err
	}

	c.portsMu.Lock()
	defer c.portsMu.Unlock()

	if c.m == nil {
		_ = newMon.Close()
		newCmd.close()
		return fmt.Errorf("monitor missing during reconnect")
	}

	c.c = newCmd
	c.m.r = newMon
	return nil
}

func (c *Controller) withCommand(fn func(*command) error) error {
	if c == nil {
		return fmt.Errorf("controller not connected")
	}
	c.portsMu.RLock()
	cmd := c.c
	c.portsMu.RUnlock()
	if cmd == nil {
		return fmt.Errorf("controller not connected")
	}
	return fn(cmd)
}

func (c *Controller) monitorRadioPort() *icom.Radio {
	if c == nil || c.m == nil {
		return nil
	}
	c.portsMu.RLock()
	defer c.portsMu.RUnlock()
	return c.m.r
}

func openMonitorRadio() (*icom.Radio, error) {
	r, err := icom.OpenRadio(config.Radio.MonitorPort, config.Radio.Baud, config.Radio.Address)
	if err != nil {
		return nil, wrapOpenError("radio monitor port", config.Radio.MonitorPort, err)
	}
	return r, nil
}

func nextBackoff(d time.Duration) time.Duration {
	d *= 2
	if d > reconnectBackoffMax {
		return reconnectBackoffMax
	}
	return d
}

func sleepQuit(d time.Duration, quit <-chan bool) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-quit:
		return false
	case <-t.C:
		return true
	}
}

func (c *Controller) isReconnecting() bool {
	return c != nil && c.reconnecting.Load()
}
