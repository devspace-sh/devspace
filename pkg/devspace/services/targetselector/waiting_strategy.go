package targetselector

import (
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
)

// WaitingStrategy defines how the target selector should wait
type WaitingStrategy interface {
	SelectContainer(containers []*selector.SelectedPodContainer, log log.Logger) (bool, *selector.SelectedPodContainer, error)
	SelectPod(pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error)
}
