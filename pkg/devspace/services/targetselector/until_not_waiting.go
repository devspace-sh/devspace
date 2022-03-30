package targetselector

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"sort"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
)

// NewUntilNotWaitingStrategy creates a new waiting strategy
func NewUntilNotWaitingStrategy(initialDelay time.Duration) WaitingStrategy {
	return &untilNotWaiting{
		originalDelay: initialDelay,
		initialDelay:  time.Now().Add(initialDelay),
		podInfoPrinter: &PodInfoPrinter{
			LastWarning: time.Now().Add(initialDelay),
		},
	}
}

// this waiting strategy will wait until the newest pod / container is not in waiting
// stage anymore and either terminated or running.
type untilNotWaiting struct {
	originalDelay time.Duration
	initialDelay  time.Time

	podInfoPrinter *PodInfoPrinter
}

func (u *untilNotWaiting) Reset() WaitingStrategy {
	return &untilNotWaiting{
		originalDelay: u.originalDelay,
		initialDelay:  time.Now().Add(u.originalDelay),
		podInfoPrinter: &PodInfoPrinter{
			LastWarning: time.Now().Add(u.originalDelay),
		},
	}
}

func (u *untilNotWaiting) SelectPod(ctx context.Context, client kubectl.Client, namespace string, pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error) {
	now := time.Now()
	if now.Before(u.initialDelay) {
		return false, nil, nil
	} else if len(pods) == 0 {
		if now.After(u.initialDelay.Add(time.Second * 6)) {
			u.podInfoPrinter.PrintNotFoundWarning(ctx, client, namespace, log)
		}

		return false, nil, nil
	}

	sort.Slice(pods, func(i, j int) bool {
		return selector.SortPodsByNewest(pods, i, j)
	})
	if isPodWaiting(pods[0]) {
		return false, nil, nil
	}

	return true, pods[0], nil
}

func (u *untilNotWaiting) SelectContainer(ctx context.Context, client kubectl.Client, namespace string, containers []*selector.SelectedPodContainer, log log.Logger) (bool, *selector.SelectedPodContainer, error) {
	now := time.Now()
	if now.Before(u.initialDelay) {
		return false, nil, nil
	} else if len(containers) == 0 {
		if now.After(u.initialDelay.Add(time.Second * 6)) {
			u.podInfoPrinter.PrintNotFoundWarning(ctx, client, namespace, log)
		}

		return false, nil, nil
	}

	sort.Slice(containers, func(i, j int) bool {
		return selector.SortContainersByNewest(containers, i, j)
	})
	if isContainerWaiting(containers[0]) {
		return false, nil, nil
	}

	return true, containers[0], nil
}

func isPodWaiting(pod *v1.Pod) bool {
	if selector.IsPodTerminating(pod) {
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

func isContainerWaiting(container *selector.SelectedPodContainer) bool {
	if selector.IsPodTerminating(container.Pod) {
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
