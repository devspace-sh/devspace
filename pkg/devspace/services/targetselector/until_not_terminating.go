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

// NewUntilNotTerminatingStrategy creates a new waiting strategy
func NewUntilNotTerminatingStrategy(initialDelay time.Duration) WaitingStrategy {
	return &untilNotTerminating{
		originalDelay: initialDelay,
		initialDelay:  time.Now().Add(initialDelay),
		podInfoPrinter: &PodInfoPrinter{
			LastWarning: time.Now().Add(initialDelay),
		},
	}
}

// this waiting strategy will wait until there is a pod that is not terminating
type untilNotTerminating struct {
	originalDelay time.Duration
	initialDelay  time.Time

	podInfoPrinter *PodInfoPrinter
}

func (u *untilNotTerminating) Reset() WaitingStrategy {
	return &untilNotTerminating{
		originalDelay: u.originalDelay,
		initialDelay:  time.Now().Add(u.originalDelay),
		podInfoPrinter: &PodInfoPrinter{
			LastWarning: time.Now().Add(u.originalDelay),
		},
	}
}

func (u *untilNotTerminating) SelectPod(ctx context.Context, client kubectl.Client, namespace string, pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error) {
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
	if selector.IsPodTerminating(pods[0]) {
		return false, nil, nil
	}

	return true, pods[0], nil
}

func (u *untilNotTerminating) SelectContainer(ctx context.Context, client kubectl.Client, namespace string, containers []*selector.SelectedPodContainer, log log.Logger) (bool, *selector.SelectedPodContainer, error) {
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
	if selector.IsPodTerminating(containers[0].Pod) {
		return false, nil, nil
	}

	return true, containers[0], nil
}
