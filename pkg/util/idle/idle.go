package idle

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/kill"
	"time"

	"github.com/loft-sh/devspace/pkg/util/log"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Monitor interface {
	Start(timeout time.Duration, log log.Logger)
}

type Getter interface {
	Idle() (time.Duration, error)
}

// NewIdleMonitor returns a new idle monitor
func NewIdleMonitor() (Monitor, error) {
	getter, err := NewIdleGetter()
	if err != nil {
		if _, ok := err.(*unsupportedError); ok {
			return nil, nil
		}

		return nil, err
	}

	return &monitor{
		Getter: getter,
	}, nil
}

type monitor struct {
	Getter Getter
}

func (m *monitor) Start(timeout time.Duration, log log.Logger) {
	if timeout <= 0 {
		return
	}

	go func() {
		wait.Forever(func() {
			duration, err := m.Getter.Idle()
			if err != nil {
				// don't do anything
				return
			} else if duration > timeout {
				// we exit here
				kill.StopDevSpace(fmt.Sprintf("Automatically exit DevSpace, because the user is inactive for %s. To disable automatic exiting, run with --inactivity-timeout=0", duration.String()))
			}
		}, time.Second*10)
	}()
}

type unsupportedError struct{}

func (u *unsupportedError) Error() string {
	return "not supported"
}
