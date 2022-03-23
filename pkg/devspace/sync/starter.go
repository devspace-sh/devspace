package sync

import "sync"

type DelayedContainerStarter interface {
	Inc()
	Done(start func() error) error
}

func NewDelayedContainerStarter() DelayedContainerStarter {
	return &delayedContainerStarter{
		amount: 0,
	}
}

type delayedContainerStarter struct {
	m      sync.Mutex
	amount int
}

func (d *delayedContainerStarter) Inc() {
	d.m.Lock()
	defer d.m.Unlock()

	d.amount++
}

func (d *delayedContainerStarter) Done(start func() error) error {
	d.m.Lock()
	defer d.m.Unlock()

	if d.amount <= 0 {
		return nil
	}

	d.amount--
	if d.amount == 0 {
		return start()
	}

	return nil
}
