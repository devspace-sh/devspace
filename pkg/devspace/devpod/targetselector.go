package devpod

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	corev1 "k8s.io/api/core/v1"
	"sort"
	"time"
)

type DevPodLostConnection struct{}

func (d DevPodLostConnection) Error() string {
	return "lost connection to pod"
}

func newTargetSelector(pod, namespace, defaultContainer string, parent *tomb.Tomb) targetselector.TargetSelector {
	return &targetSelector{
		pod:              pod,
		namespace:        namespace,
		defaultContainer: defaultContainer,
		parent:           parent,
	}
}

type targetSelector struct {
	pod              string
	namespace        string
	defaultContainer string
	container        string

	// parent is killed if we cannot find the
	// pod anymore we are assigned to
	parent *tomb.Tomb
}

func (t *targetSelector) SelectSinglePod(ctx context.Context, client kubectl.Client, log log.Logger) (*corev1.Pod, error) {
	options := targetselector.NewEmptyOptions().
		WithPod(t.pod).
		WithNamespace(t.namespace).
		WithWaitingStrategy(newUntilNewestRunningWaitingStrategy(time.Millisecond*250, t.parent))

	return targetselector.NewTargetSelector(options).SelectSinglePod(ctx, client, log)
}

func (t *targetSelector) SelectSingleContainer(ctx context.Context, client kubectl.Client, log log.Logger) (*selector.SelectedPodContainer, error) {
	container := t.container
	if t.container == "" {
		container = t.defaultContainer
	}

	options := targetselector.NewEmptyOptions().
		WithPod(t.pod).
		WithNamespace(t.namespace).
		WithContainer(container).
		WithWaitingStrategy(newUntilNewestRunningWaitingStrategy(time.Millisecond*250, t.parent))

	return targetselector.NewTargetSelector(options).SelectSingleContainer(ctx, client, log)
}

func (t *targetSelector) WithContainer(container string) targetselector.TargetSelector {
	return &targetSelector{
		pod:              t.pod,
		namespace:        t.namespace,
		container:        container,
		defaultContainer: t.defaultContainer,
		parent:           t.parent,
	}
}

// newUntilNewestRunningWaitingStrategy creates a new waiting strategy
func newUntilNewestRunningWaitingStrategy(delay time.Duration, parent *tomb.Tomb) targetselector.WaitingStrategy {
	return &untilNewestRunning{
		originalDelay: delay,
		initialDelay:  time.Now().Add(delay),
		parent:        parent,
		podInfoPrinter: &targetselector.PodInfoPrinter{
			LastWarning: time.Now().Add(delay),
		},
	}
}

// this waiting strategy will wait until the newest pod / container is up and running or fails
// it also waits initially for some time
type untilNewestRunning struct {
	originalDelay time.Duration
	initialDelay  time.Time

	podInfoPrinter *targetselector.PodInfoPrinter

	// parent is killed if we cannot find the
	// pod anymore we are assigned to
	parent *tomb.Tomb
}

func (u *untilNewestRunning) Reset() targetselector.WaitingStrategy {
	return &untilNewestRunning{
		parent:        u.parent,
		originalDelay: u.originalDelay,
		initialDelay:  time.Now().Add(u.originalDelay),
		podInfoPrinter: &targetselector.PodInfoPrinter{
			LastWarning: time.Now().Add(u.originalDelay),
		},
	}
}

func (u *untilNewestRunning) SelectPod(ctx context.Context, client kubectl.Client, namespace string, pods []*corev1.Pod, log log.Logger) (bool, *corev1.Pod, error) {
	now := time.Now()
	if now.Before(u.initialDelay) {
		return false, nil, nil
	} else if len(pods) == 0 {
		u.parent.Kill(DevPodLostConnection{})
		return false, nil, DevPodLostConnection{}
	}

	sort.Slice(pods, func(i, j int) bool {
		return selector.SortPodsByNewest(pods, i, j)
	})
	if targetselector.HasPodProblem(pods[0]) {
		u.podInfoPrinter.PrintPodWarning(pods[0], log)
		return false, nil, nil
	} else if kubectl.GetPodStatus(pods[0]) != "Running" {
		u.podInfoPrinter.PrintPodInfo(ctx, client, pods[0], log)
		return false, nil, nil
	}

	return true, pods[0], nil
}

func (u *untilNewestRunning) SelectContainer(ctx context.Context, client kubectl.Client, namespace string, containers []*selector.SelectedPodContainer, log log.Logger) (bool, *selector.SelectedPodContainer, error) {
	now := time.Now()
	if now.Before(u.initialDelay) {
		return false, nil, nil
	} else if len(containers) == 0 {
		u.parent.Kill(DevPodLostConnection{})
		return false, nil, DevPodLostConnection{}
	}

	sort.Slice(containers, func(i, j int) bool {
		return selector.SortContainersByNewest(containers, i, j)
	})
	if targetselector.HasPodProblem(containers[0].Pod) {
		u.podInfoPrinter.PrintPodWarning(containers[0].Pod, log)
		return false, nil, nil
	} else if !targetselector.IsContainerRunning(containers[0]) {
		u.podInfoPrinter.PrintPodInfo(ctx, client, containers[0].Pod, log)
		return false, nil, nil
	}

	return true, containers[0], nil
}
