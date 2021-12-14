package targetselector

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
)

// NewUntilNewestRunningWaitingStrategy creates a new waiting strategy
func NewUntilNewestRunningWaitingStrategy(initialDelay time.Duration) WaitingStrategy {
	return &untilNewestRunning{
		initialDelay: time.Now().Add(initialDelay),
		lastWarning:  time.Now().Add(initialDelay),
	}
}

// this waiting strategy will wait until the newest pod / container is up and running or fails
// it also waits initially for some time
type untilNewestRunning struct {
	initialDelay time.Time

	lastMutex   sync.Mutex
	lastWarning time.Time
}

func (u *untilNewestRunning) SelectPod(pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error) {
	now := time.Now()
	if now.Before(u.initialDelay) {
		return false, nil, nil
	} else if len(pods) == 0 {
		u.printNotFoundWarning(log)
		return false, nil, nil
	}

	sort.Slice(pods, func(i, j int) bool {
		return selector.SortPodsByNewest(pods, i, j)
	})
	if HasPodProblem(pods[0]) {
		u.printPodWarning(pods[0], log)
		return false, nil, nil
	} else if kubectl.GetPodStatus(pods[0]) != "Running" {
		u.printPodInfo(pods[0], log)
		return false, nil, nil
	}

	return true, pods[0], nil
}

func (u *untilNewestRunning) SelectContainer(containers []*selector.SelectedPodContainer, log log.Logger) (bool, *selector.SelectedPodContainer, error) {
	now := time.Now()
	if now.Before(u.initialDelay) {
		return false, nil, nil
	} else if len(containers) == 0 {
		u.printNotFoundWarning(log)
		return false, nil, nil
	}

	sort.Slice(containers, func(i, j int) bool {
		return selector.SortContainersByNewest(containers, i, j)
	})
	if HasPodProblem(containers[0].Pod) {
		u.printPodWarning(containers[0].Pod, log)
		return false, nil, nil
	} else if !IsContainerRunning(containers[0]) {
		u.printPodInfo(containers[0].Pod, log)
		return false, nil, nil
	}

	return true, containers[0], nil
}

func (u *untilNewestRunning) printPodInfo(pod *v1.Pod, log log.Logger) {
	u.lastMutex.Lock()
	defer u.lastMutex.Unlock()

	if time.Since(u.lastWarning) > time.Second*10 {
		status := kubectl.GetPodStatus(pod)
		log.Warnf("DevSpace is waiting, because Pod %s/%s has status: %s", pod.Namespace, pod.Name, status)
		u.lastWarning = time.Now()
	}
}

func (u *untilNewestRunning) printNotFoundWarning(log log.Logger) {
	u.lastMutex.Lock()
	defer u.lastMutex.Unlock()

	if time.Since(u.lastWarning) > time.Second*10 {
		log.Warnf("DevSpace still couldn't find any Pods that match the selector. DevSpace will continue waiting, but this operation might timeout")
		u.lastWarning = time.Now()
	}
}

func (u *untilNewestRunning) printPodWarning(pod *v1.Pod, log log.Logger) {
	u.lastMutex.Lock()
	defer u.lastMutex.Unlock()

	if time.Since(u.lastWarning) > time.Second*10 {
		status := kubectl.GetPodStatus(pod)
		log.Warnf("Pod %s/%s has critical status: %s. DevSpace will continue waiting, but this operation might timeout", pod.Namespace, pod.Name, status)
		u.lastWarning = time.Now()
	}
}

func IsContainerRunning(container *selector.SelectedPodContainer) bool {
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
	status = strings.TrimPrefix(status, "Init:")
	return kubectl.CriticalStatus[status]
}
