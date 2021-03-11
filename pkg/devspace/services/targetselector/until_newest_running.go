package targetselector

import (
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
	"sort"
	"strings"
	"sync"
	"time"
)

// NewUntilNewestRunningWaitingStrategy creates a new waiting strategy
func NewUntilNewestRunningWaitingStrategy(initialDelay time.Duration) WaitingStrategy {
	return &untilNewestRunning{
		initialDelay: time.Now().Add(initialDelay),
	}
}

// this waiting strategy will wait until the newest pod / container is up and running or fails
// it also waits initially for some time
type untilNewestRunning struct {
	initialDelay    time.Time
	printWarning    sync.Once
	notFoundWarning sync.Once
}

func (u *untilNewestRunning) SelectPod(pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error) {
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
	if HasPodProblem(pods[0]) {
		u.printPodWarning(pods[0], log)
		return false, nil, nil
	}
	if kubectl.GetPodStatus(pods[0]) != "Running" {
		return false, nil, nil
	}

	return true, pods[0], nil
}

func (u *untilNewestRunning) SelectContainer(containers []*kubectl.SelectedPodContainer, log log.Logger) (bool, *kubectl.SelectedPodContainer, error) {
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
	if HasPodProblem(containers[0].Pod) {
		u.printPodWarning(containers[0].Pod, log)
		return false, nil, nil
	}
	if IsContainerRunning(containers[0]) == false {
		return false, nil, nil
	}

	return true, containers[0], nil
}

func (u *untilNewestRunning) printNotFoundWarning(log log.Logger) {
	u.notFoundWarning.Do(func() {
		log.Warnf("DevSpace still couldn't find any Pods that match the selector. DevSpace will continue waiting, but this operation might timeout")
	})
}

func (u *untilNewestRunning) printPodWarning(pod *v1.Pod, log log.Logger) {
	u.printWarning.Do(func() {
		status := kubectl.GetPodStatus(pod)
		log.Warnf("Pod %s/%s has critical status: %s. DevSpace will continue waiting, but this operation might timeout", pod.Namespace, pod.Name, status)
	})
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

func HasPodProblem(pod *v1.Pod) bool {
	status := kubectl.GetPodStatus(pod)
	if strings.HasPrefix(status, "Init:") {
		status = status[5:]
	}

	return kubectl.CriticalStatus[status]
}
