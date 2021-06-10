package controller

import "github.com/bbathe/icom-powercombo-controller/status"

type Controller struct {
	c *command
	m *monitor
}

var (
	controller *Controller
)

func NewController() *Controller {
	if controller == nil {
		controller = new(Controller)

		c := newCommand()
		controller.c = c

		m := newMonitor()
		controller.m = m
	}

	return controller
}

func (c *Controller) Close() {
	c.m.close()
	c.c.close()

	status.SetStatuses(status.StatusUnknown)

	controller = nil
}

// SetKPA500Mode exposes setting the KPA500 mode (operate/standby) to the UI
func (c *Controller) SetKPA500Mode(mode int) error {
	return c.c.setKPA500Mode(mode)
}

// KAT500FullTune initiates a full tune on the KAT500
func (c *Controller) KAT500FullTune() error {
	return c.c.KAT500FullTune()
}
