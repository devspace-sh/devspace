package pipeline

import (
	"context"
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"sync"
)

type Job interface {
	Start(ctx *devspacecontext.Context, workload func(ctx *devspacecontext.Context) error) error
	Stop()
	Done() <-chan struct{}
	Error() error
}

type job struct {
	allowRestart bool

	err error

	stopped bool

	doneMutex sync.Mutex
	done      chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
}

func NewJob(allowRestart bool) Job {
	return &job{
		allowRestart: allowRestart,
		done:         make(chan struct{}),
	}
}

func (j *job) Error() error {
	j.doneMutex.Lock()
	defer j.doneMutex.Unlock()

	return j.err
}

// Stop signals the job to stop but does not wait
// until it is actually stopped, it just cancels the
// context the job is using to do its things
func (j *job) Stop() {
	j.doneMutex.Lock()
	defer j.doneMutex.Unlock()

	if j.cancel != nil {
		j.cancel()
	}
}

func (j *job) Done() <-chan struct{} {
	j.doneMutex.Lock()
	defer j.doneMutex.Unlock()

	return j.done
}

func (j *job) Start(ctx *devspacecontext.Context, workload func(ctx *devspacecontext.Context) error) error {
	j.doneMutex.Lock()
	defer j.doneMutex.Unlock()

	if j.cancel != nil {
		if !j.allowRestart {
			return fmt.Errorf("restart not allowed")
		}

		j.cancel()
		<-j.done
	}

	j.err = nil
	j.done = make(chan struct{})

	j.ctx, j.cancel = context.WithCancel(ctx.Context)
	ctx = ctx.WithContext(j.ctx)
	go func() {
		j.stopWithErr(workload(ctx))
	}()
	return nil
}

func (j *job) stopWithErr(err error) {
	j.doneMutex.Lock()
	defer j.doneMutex.Unlock()

	j.cancel()
	j.ctx = nil
	j.err = err
	j.stopped = true
	close(j.done)
}
