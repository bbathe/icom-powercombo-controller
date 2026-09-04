package controller

import (
	"fmt"
)

type jobKind int

const (
	jobNormal jobKind = iota
	jobPollKAT
	jobPollKPA
)

type commandJob struct {
	kind jobKind
	fn   func(*command) error
	res  chan error
}

func (c *Controller) startCommandWorker() {
	c.jobs = make(chan commandJob, 16)
	c.workerQuit = make(chan struct{})
	c.workerDone = make(chan struct{})
	go c.commandWorker()
}

func (c *Controller) stopCommandWorker() {
	if c.workerQuit == nil {
		return
	}

	select {
	case <-c.workerQuit:
		// already closed
	default:
		close(c.workerQuit)
	}

	if c.workerDone != nil {
		<-c.workerDone
	}

	// unblock any waiters still sitting in the queue
	for {
		select {
		case j := <-c.jobs:
			c.clearPollPending(j.kind)
			if j.res != nil {
				j.res <- fmt.Errorf("controller shutting down")
			}
		default:
			return
		}
	}
}

func (c *Controller) commandWorker() {
	defer close(c.workerDone)

	for {
		select {
		case <-c.workerQuit:
			return
		case j := <-c.jobs:
			c.portsMu.RLock()
			cmd := c.c
			c.portsMu.RUnlock()

			var err error
			if cmd == nil {
				err = fmt.Errorf("controller not connected")
			} else {
				err = j.fn(cmd)
			}

			c.clearPollPending(j.kind)
			if j.res != nil {
				j.res <- err
			}
		}
	}
}

func (c *Controller) clearPollPending(kind jobKind) {
	switch kind {
	case jobPollKAT:
		c.pollKATPending.Store(false)
	case jobPollKPA:
		c.pollKPAPending.Store(false)
	}
}

func (c *Controller) enqueue(kind jobKind, fn func(*command) error) error {
	if c == nil || c.jobs == nil {
		return fmt.Errorf("controller not connected")
	}

	res := make(chan error, 1)
	job := commandJob{kind: kind, fn: fn, res: res}

	select {
	case <-c.workerQuit:
		c.clearPollPending(kind)
		return fmt.Errorf("controller shutting down")
	case c.jobs <- job:
	}

	select {
	case <-c.workerQuit:
		select {
		case err := <-res:
			return err
		default:
			c.clearPollPending(kind)
			return fmt.Errorf("controller shutting down")
		}
	case err := <-res:
		return err
	}
}

func (c *Controller) withCommand(fn func(*command) error) error {
	return c.enqueue(jobNormal, fn)
}

// tryPollKAT enqueues a KAT poll, or no-ops if one is already pending/in flight.
func (c *Controller) tryPollKAT(fn func(*command) error) error {
	if c == nil || !c.pollKATPending.CompareAndSwap(false, true) {
		return nil
	}
	return c.enqueue(jobPollKAT, fn)
}

// tryPollKPA enqueues a KPA poll, or no-ops if one is already pending/in flight.
func (c *Controller) tryPollKPA(fn func(*command) error) error {
	if c == nil || !c.pollKPAPending.CompareAndSwap(false, true) {
		return nil
	}
	return c.enqueue(jobPollKPA, fn)
}
