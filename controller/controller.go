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
}

var (
	controller *Controller
)

func NewController() (*Controller, error) {
	if controller != nil {
		return controller, nil
	}

	c, err := newCommand()
	if err != nil {
		return nil, err
	}

	// monitor init and polls use the package singleton's command side
	ctrl := &Controller{c: c}
	controller = ctrl

	m, err := newMonitor()
	if err != nil {
		c.close()
		controller = nil
		return nil, err
	}

	ctrl.m = m
	return ctrl, nil
}

func (c *Controller) Close() {
	if c == nil {
		return
	}

	if c.m != nil {
		c.m.close()
	}

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

	controller = nil
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
	c.m.trackKAT500 = t
}

// wrapOpenError formats a device open failure for the UI
func wrapOpenError(device, port string, err error) error {
	return fmt.Errorf("%s open failed (%s): %w", device, port, err)
}
