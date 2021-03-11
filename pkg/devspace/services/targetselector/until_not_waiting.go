package targetselector

import (
	kubectl "github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
	"sort"
	"sync"
	"time"
)

// NewUntilNotWaitingStrategy creates a new waiting strategy
func NewUntilNotWaitingStrategy(initialDelay time.Duration) WaitingStrategy {
	return &untilNotWaiting{
		initialDelay: time.Now().Add(initialDelay),
	}
}

// this waiting strategy will wait until the newest pod / container is not in waiting
// stage anymore and either terminated or running.
type untilNotWaiting struct {
	initialDelay    time.Time
	notFoundWarning sync.Once
}

func (u *untilNotWaiting) SelectPod(pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error) {
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
		return kubectl.SortPodsByNewest(pods, i, j)
	})
	if isPodWaiting(pods[0]) {
		return false, nil, nil
	}

	return true, pods[0], nil
}

func (u *untilNotWaiting) SelectContainer(containers []*kubectl.SelectedPodContainer, log log.Logger) (bool, *kubectl.SelectedPodContainer, error) {
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
		return kubectl.SortContainersByNewest(containers, i, j)
	})
	if isContainerWaiting(containers[0]) {
		return false, nil, nil
	}

	return true, containers[0], nil
}

func (u *untilNotWaiting) printNotFoundWarning(log log.Logger) {
	u.notFoundWarning.Do(func() {
		log.Warnf("DevSpace still couldn't find any Pods that match the selector. DevSpace will continue waiting, but this operation might timeout")
	})
}

func isPodWaiting(pod *v1.Pod) bool {
	if pod.DeletionTimestamp != nil {
		return true
	}
	
	for _, containerStatus := range pod.Status.InitContainerStatuses {
		if containerStatus.State.Waiting != nil {
			return true
		}
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil {
			return true
		}
	}

	return false
}

func isContainerWaiting(container *kubectl.SelectedPodContainer) bool {
	if container.Pod.DeletionTimestamp != nil {
		return true
	}
	
	for _, containerStatus := range container.Pod.Status.InitContainerStatuses {
		if containerStatus.Name == container.Container.Name && containerStatus.State.Waiting != nil {
			return true
		}
	}
	for _, containerStatus := range container.Pod.Status.ContainerStatuses {
		if containerStatus.Name == container.Container.Name && containerStatus.State.Waiting != nil {
			return true
		}
	}

	return false
}
