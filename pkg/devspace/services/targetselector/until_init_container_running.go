package targetselector

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
)

// NewUntilInitContainerIsRunningWaitingStrategy creates a new waiting strategy
func NewUntilInitContainerIsRunningWaitingStrategy(delay time.Duration, devPod *latest.DevPod) WaitingStrategy {
	return &untilInitContainerIsNewestRunning{
		originalDelay: delay,
		initialDelay:  time.Now().Add(delay),
		podInfoPrinter: &PodInfoPrinter{
			LastWarning: time.Now().Add(delay),
		},
		devPod: devPod,
	}
}

// this waiting strategy will wait until the newest init container is up and running or fails
type untilInitContainerIsNewestRunning struct {
	originalDelay time.Duration
	initialDelay  time.Time

	devPod *latest.DevPod

	podInfoPrinter *PodInfoPrinter
}

func (u *untilInitContainerIsNewestRunning) Reset() WaitingStrategy {
	return &untilInitContainerIsNewestRunning{
		originalDelay: u.originalDelay,
		initialDelay:  time.Now().Add(u.originalDelay),
		podInfoPrinter: &PodInfoPrinter{
			LastWarning: time.Now().Add(u.originalDelay),
		},
		devPod: u.devPod,
	}
}

func (u *untilInitContainerIsNewestRunning) SelectPod(ctx context.Context, client kubectl.Client, namespace string, pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error) {
	return true, nil, nil
}

func (u *untilInitContainerIsNewestRunning) SelectContainer(ctx context.Context, client kubectl.Client, namespace string, containers []*selector.SelectedPodContainer, log log.Logger) (bool, *selector.SelectedPodContainer, error) {
	now := time.Now()
	if now.Before(u.initialDelay) {
		return false, nil, nil
	} else if len(containers) == 0 {
		u.podInfoPrinter.PrintNotFoundWarning(ctx, client, namespace, log)
		return false, nil, nil
	}

	if u.devPod.LabelSelector != nil && u.devPod.Container == "" && u.devPod.Containers == nil {
		return false, nil, fmt.Errorf("with dev.LabelSelector use dev.container or dev.containers to select the container")
	}

	var initContainerName string
	if u.devPod.LabelSelector != nil {
		if u.devPod.Container != "" {
			initContainerName = u.devPod.Container
		} else {
			for _, container := range u.devPod.Containers {
				if container.InitContainer {
					initContainerName = container.Container
					break
				}
			}
		}

		for _, container := range containers {
			if container.Container.Name == initContainerName {
				if !isInitContainerRunning(container) {
					return false, nil, nil
				}
				return true, container, nil
			}
		}
	}

	if u.devPod.ImageSelector != "" {
		for _, container := range containers {
			if u.devPod.DevImage != "" {
				if u.devPod.DevImage == container.Container.Image {
					if !isInitContainerRunning(container) {
						return false, nil, nil
					}
					return true, container, nil
				}
			}

			if u.devPod.ImageSelector == container.Container.Image {
				if !isInitContainerRunning(container) {
					return false, nil, nil
				}
				return true, container, nil
			}
		}
	}
	return false, nil, nil
}

func isInitContainerRunning(container *selector.SelectedPodContainer) bool {
	if selector.IsPodTerminating(container.Pod) {
		return false
	}
	for _, cs := range container.Pod.Status.InitContainerStatuses {
		if cs.Name == container.Container.Name && !cs.Ready && cs.State.Running != nil {
			return true
		}
	}
	return false
}
