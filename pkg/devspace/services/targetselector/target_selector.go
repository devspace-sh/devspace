package targetselector

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DefaultPodQuestion defines the default question for selecting a pod
const DefaultPodQuestion = "Select a pod"

// DefaultContainerQuestion defines the default question for selecting a container
const DefaultContainerQuestion = "Select a container"

// TargetSelector is the struct that will select a target
type TargetSelector struct {
	PodQuestion       *string
	ContainerQuestion *string

	namespace string
	pick      bool

	labelSelector *string
	podName       *string
	containerName *string

	allowPick bool
	config    *latest.Config
}

// NewTargetSelector creates a new target selector for selecting a target pod or container
func NewTargetSelector(config *latest.Config, sp *SelectorParameter, allowPick bool) (*TargetSelector, error) {
	// Get namespace
	namespace, err := sp.GetNamespace(config)
	if err != nil {
		return nil, err
	}

	// Get label selector
	labelSelector, err := sp.GetLabelSelector(config)
	if err != nil {
		return nil, err
	}

	return &TargetSelector{
		namespace:     namespace,
		labelSelector: labelSelector,
		podName:       sp.GetPodName(),
		containerName: sp.GetContainerName(),
		pick:          allowPick && sp.CmdParameter.Pick != nil && *sp.CmdParameter.Pick == true,

		allowPick: allowPick,
		config:    config,
	}, nil
}

// GetPod retrieves a pod
func (t *TargetSelector) GetPod(client kubernetes.Interface) (*v1.Pod, error) {
	if t.pick == false && t.podName != nil {
		pod, err := client.Core().Pods(t.namespace).Get(*t.podName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		podStatus := kubectl.GetPodStatus(pod)
		if podStatus != "Running" && strings.HasPrefix(podStatus, "Init") == false {
			return nil, fmt.Errorf("Couldn't get pod %s, because pod has status: %s which is not Running", pod.Name, podStatus)
		}

		return pod, nil
	} else if t.pick == false && t.labelSelector != nil {
		pod, err := kubectl.GetNewestRunningPod(t.config, client, *t.labelSelector, t.namespace, time.Second*120)
		if err != nil {
			return nil, err
		}

		return pod, nil
	}

	// Don't allow pick
	if t.allowPick == false {
		return nil, errors.New("Couldn't find a running pod, because no labelselector or pod name was specified")
	}

	// Ask for pod
	pod, err := SelectPod(client, t.namespace, nil, t.PodQuestion)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

// GetContainer retrieves a container and pod
func (t *TargetSelector) GetContainer(client kubernetes.Interface) (*v1.Pod, *v1.Container, error) {
	pod, err := t.GetPod(client)
	if err != nil {
		return nil, nil, err
	}
	if pod == nil {
		return nil, nil, fmt.Errorf("Couldn't find a running pod in namespace %s", t.namespace)
	}

	// Select container if necessary
	if pod.Spec.Containers != nil && len(pod.Spec.Containers) == 1 {
		return pod, &pod.Spec.Containers[0], nil
	} else if pod.Spec.Containers != nil && len(pod.Spec.Containers) > 1 {
		if t.pick == false && t.containerName != nil {
			// Find container
			for _, container := range pod.Spec.Containers {
				if container.Name == *t.containerName {
					return pod, &container, nil
				}
			}

			return nil, nil, fmt.Errorf("Couldn't find container %s in pod %s", *t.containerName, pod.Name)
		}

		// Don't allow pick
		if t.allowPick == false {
			return nil, nil, fmt.Errorf("Couldn't select a container in pod %s, because no container name was specified", pod.Name)
		}

		options := []string{}
		for _, container := range pod.Spec.Containers {
			options = append(options, container.Name)
		}

		if t.ContainerQuestion == nil {
			t.ContainerQuestion = ptr.String(DefaultContainerQuestion)
		}

		containerName := survey.Question(&survey.QuestionOptions{
			Question: *t.ContainerQuestion,
			Options:  options,
		})
		for _, container := range pod.Spec.Containers {
			if container.Name == containerName {
				return pod, &container, nil
			}
		}
	}

	return pod, nil, nil
}
