package targetselector

import (
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
	"sort"
	"sync"
	"time"
)

// NewUntilNotTerminatingStrategy creates a new waiting strategy
func NewUntilNotTerminatingStrategy(initialDelay time.Duration) WaitingStrategy {
	return &untilNotTerminating{
		initialDelay: time.Now().Add(initialDelay),
	}
}

// this waiting strategy will wait until there is a pod that is not terminating
type untilNotTerminating struct {
	initialDelay    time.Time
	notFoundWarning sync.Once
}

func (u *untilNotTerminating) SelectPod(pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error) {
	now := time.Now()
	if now.Before(u.initialDelay) {
		return false, nil, nil
	} else if len(pods) == 0 {
		if now.After(u.initialDelay.Add(time.Second * 6)) {
			u.printNotFoundWarning(log)
		}

		return false, nil, nil
	}

	sort.Slice(pods, func(i, j int) bool {
		return selector.SortPodsByNewest(pods, i, j)
	})
	if isPodTerminating(pods[0]) {
		return false, nil, nil
	}

	return true, pods[0], nil
}

func (u *untilNotTerminating) SelectContainer(containers []*selector.SelectedPodContainer, log log.Logger) (bool, *selector.SelectedPodContainer, error) {
	now := time.Now()
	if now.Before(u.initialDelay) {
		return false, nil, nil
	} else if len(containers) == 0 {
		if now.After(u.initialDelay.Add(time.Second * 6)) {
			u.printNotFoundWarning(log)
		}

		return false, nil, nil
	}

	sort.Slice(containers, func(i, j int) bool {
		return selector.SortContainersByNewest(containers, i, j)
	})
	if isPodTerminating(containers[0].Pod) {
		return false, nil, nil
	}

	return true, containers[0], nil
}

func (u *untilNotTerminating) printNotFoundWarning(log log.Logger) {
	u.notFoundWarning.Do(func() {
		log.Warnf("DevSpace still couldn't find any Pods that match the selector. DevSpace will continue waiting, but this operation might timeout")
	})
}

func isPodTerminating(pod *v1.Pod) bool {
	if pod.DeletionTimestamp != nil {
		return true
	}

	return false
}
