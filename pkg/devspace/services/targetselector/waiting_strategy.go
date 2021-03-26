package targetselector

import (
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
	v1 "k8s.io/api/core/v1"
)

// WaitingStrategy defines how the target selector should wait
type WaitingStrategy interface {
	SelectContainer(containers []*kubectl.SelectedPodContainer, log log.Logger) (bool, *kubectl.SelectedPodContainer, error)
	SelectPod(pods []*v1.Pod, log log.Logger) (bool, *v1.Pod, error)
}
