package services

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sync"
)

type Runner interface {
	Run(fn func() error) error
	Wait() error
}

func NewRunner(maxParallel int) Runner {
	return &runner{
		errs:        []error{},
		maxParallel: maxParallel,
	}
}

type runner struct {
	errs      []error
	errsMutex sync.Mutex

	waitGroup sync.WaitGroup

	maxParallel int
	started     int
}

// Run is expected to be called from a single thread
func (r *runner) Run(fn func() error) error {
	r.waitGroup.Add(1)
	r.started++
	go func() {
		defer r.waitGroup.Done()

		err := fn()
		if err != nil {
			r.errsMutex.Lock()
			r.errs = append(r.errs, err)
			r.errsMutex.Unlock()
		}
	}()

	if r.maxParallel > 0 && r.started%r.maxParallel == r.maxParallel-1 {
		r.waitGroup.Wait()
		return utilerrors.NewAggregate(r.errs)
	}

	return nil
}

func (r *runner) Wait() error {
	r.waitGroup.Wait()

	r.errsMutex.Lock()
	defer r.errsMutex.Unlock()
	return utilerrors.NewAggregate(r.errs)
}
