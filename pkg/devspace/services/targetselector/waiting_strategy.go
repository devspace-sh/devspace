package targetselector

import (
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
	"sort"
	"time"
)

// WaitingStrategy defines how the target selector should wait
type WaitingStrategy interface {
	SelectContainer(containers []*kubectl.SelectedPodContainer, log log.Logger) (bool, *kubectl.SelectedPodContainer, error)
	SelectPod(pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error)
}

// NewUntilNewestRunningWaitingStrategy creates a new waiting strategy
func NewUntilNewestRunningWaitingStrategy(initialDelay time.Duration) WaitingStrategy {
	return &untilNewestRunning{
		initialDelay: time.Now().Add(initialDelay),
	}
}

// this waiting strategy will wait until the newest pod / container is up and running or fails
// it also waits initially for some time
type untilNewestRunning struct {
	initialDelay time.Time
}

func (u *untilNewestRunning) SelectPod(pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error) {
	if time.Now().Before(u.initialDelay) {
		return false, nil, nil
	} else if len(pods) == 0 {
		return false, nil, nil
	}

	sort.Slice(pods, func(i, j int) bool {
		return kubectl.SortPodsByNewest(pods, i, j)
	})
	if kubectl.GetPodStatus(pods[0]) != "Running" {
		return false, nil, nil
	}

	return true, pods[0], nil
}

func (u *untilNewestRunning) SelectContainer(containers []*kubectl.SelectedPodContainer, log log.Logger) (bool, *kubectl.SelectedPodContainer, error) {
	if time.Now().Before(u.initialDelay) {
		return false, nil, nil
	} else if len(containers) == 0 {
		return false, nil, nil
	}

	sort.Slice(containers, func(i, j int) bool {
		return kubectl.SortContainersByNewest(containers, i, j)
	})
	if IsContainerRunning(containers[0]) == false {
		return false, nil, nil
	}

	return true, containers[0], nil
}

func IsContainerRunning(container *kubectl.SelectedPodContainer) bool {
	if container.Pod.DeletionTimestamp != nil {
		return false
	}
	for _, cs := range container.Pod.Status.InitContainerStatuses {
		if cs.Name == container.Container.Name && cs.State.Running != nil {
			return true
		}
	}
	for _, cs := range container.Pod.Status.ContainerStatuses {
		if cs.Name == container.Container.Name && cs.State.Running != nil {
			return true
		}
	}
	return false
}
