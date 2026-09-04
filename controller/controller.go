package controller

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/bbathe/icom-powercombo-controller/status"
)

type Controller struct {
	c *command
	m *monitor

	portsMu      sync.RWMutex
	reconnectMu  sync.Mutex
	reconnecting atomic.Bool

	jobs           chan commandJob
	workerQuit     chan struct{}
	workerDone     chan struct{}
	pollKATPending atomic.Bool
	pollKPAPending atomic.Bool
}

func NewController() (*Controller, error) {
	c, err := newCommand()
	if err != nil {
		return nil, err
	}

	ctrl := &Controller{c: c}
	ctrl.startCommandWorker()

	_, err = newMonitor(ctrl)
	if err != nil {
		ctrl.stopCommandWorker()
		c.close()
		return nil, err
	}

	return ctrl, nil
}

func (c *Controller) Close() {
	if c == nil {
		return
	}

	if c.m != nil {
		c.m.close()
	}

	c.stopCommandWorker()

	// wait for any in-progress reconnect to observe quit and exit
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	c.portsMu.Lock()
	cmd := c.c
	c.c = nil
	c.portsMu.Unlock()
	if cmd != nil {
		cmd.close()
	}

	status.SetStatuses(status.StatusUnknown)
}

// SetKPA500Mode exposes setting the KPA500 mode (operate/standby) to the UI
func (c *Controller) SetKPA500Mode(mode int) error {
	return c.withCommand(func(cmd *command) error {
		return cmd.setKPA500Mode(mode)
	})
}

// KAT500FullTune initiates a full tune on the KAT500
func (c *Controller) KAT500FullTune() error {
	return c.withCommand(func(cmd *command) error {
		return cmd.KAT500FullTune()
	})
}

// SetTrackKAT500 indictes whether frequency information should be sent to the KAT500
func (c *Controller) SetTrackKAT500(t bool) {
	if c == nil || c.m == nil {
		return
	}
	c.m.trackKAT500.Store(t)
}

// wrapOpenError formats a device open failure for the UI
func wrapOpenError(device, port string, err error) error {
	return fmt.Errorf("%s open failed (%s): %w", device, port, err)
}
