package targetselector

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"

	v1 "k8s.io/api/core/v1"
)

// DefaultPodQuestion defines the default question for selecting a pod
const DefaultPodQuestion = "Select a pod"

// DefaultContainerQuestion defines the default question for selecting a container
const DefaultContainerQuestion = "Select a container"

// Options holds the options for a target selector
type Options struct {
	kubectl.Selector

	AllowPick bool
	Question  string

	Wait    *bool
	Timeout int64

	FailIfMultiple bool
	SortPods       kubectl.SortPods
	SortContainers kubectl.SortContainers

	WaitingStrategy WaitingStrategy
}

func NewEmptyOptions() Options {
	return Options{}
}

func NewDefaultOptions() Options {
	return Options{
		AllowPick: true,
		Selector: kubectl.Selector{
			FilterContainer: kubectl.FilterNonRunningContainers,
		},
		SortPods:       kubectl.SortPodsByNewest,
		SortContainers: kubectl.SortContainersByNewest,
	}
}

func NewOptionsFromFlags(containerName string, labelSelector string, namespace string, pod string, allowPick bool) Options {
	return Options{
		AllowPick: allowPick,
		Selector: kubectl.Selector{
			LabelSelector:      labelSelector,
			Pod:                pod,
			ContainerName:      containerName,
			Namespace:          namespace,
			SkipInitContainers: false,
			FilterContainer:    kubectl.FilterNonRunningContainers,
		},
		SortPods:       kubectl.SortPodsByNewest,
		SortContainers: kubectl.SortContainersByNewest,
	}
}

func (o Options) ApplyConfigParameter(labelSelector map[string]string, namespace string, containerName string, pod string) Options {
	newOptions := o
	if containerName != "" && o.ContainerName == "" {
		newOptions.ContainerName = containerName
	}
	if labelSelector != nil && o.LabelSelector == "" {
		newOptions.LabelSelector = labels.Set(labelSelector).String()
	}
	if namespace != "" && o.Namespace == "" {
		newOptions.Namespace = namespace
	}
	if pod != "" && o.Pod == "" {
		newOptions.Pod = pod
	}
	return newOptions
}

func (o Options) ApplyCmdParameter(containerName string, labelSelector string, namespace string, pod string) Options {
	newOptions := o
	if containerName != "" {
		newOptions.ContainerName = containerName
	}
	if labelSelector != "" {
		newOptions.LabelSelector = labelSelector
	}
	if namespace != "" {
		newOptions.Namespace = namespace
	}
	if pod != "" {
		newOptions.Pod = pod
	}
	return newOptions
}

type TargetSelector interface {
	SelectSinglePod(ctx context.Context, options Options, log log.Logger) (*v1.Pod, error)
	SelectSingleContainer(ctx context.Context, options Options, log log.Logger) (*kubectl.SelectedPodContainer, error)
}

// targetSelector is the struct that will select a target
type targetSelector struct {
	client kubectl.Client
}

// NewTargetSelector creates a new target selector for selecting a target pod or container
func NewTargetSelector(client kubectl.Client) TargetSelector {
	return &targetSelector{
		client: client,
	}
}

func (t *targetSelector) SelectSingleContainer(ctx context.Context, options Options, log log.Logger) (*kubectl.SelectedPodContainer, error) {
	container, err := t.selectSingle(ctx, options, log, t.selectSingleContainer)
	if err != nil {
		return nil, err
	} else if container == nil {
		return nil, nil
	}

	return container.(*kubectl.SelectedPodContainer), nil
}

func (t *targetSelector) SelectSinglePod(ctx context.Context, options Options, log log.Logger) (*v1.Pod, error) {
	pod, err := t.selectSingle(ctx, options, log, t.selectSinglePod)
	if err != nil {
		return nil, err
	} else if pod == nil {
		return nil, nil
	}

	return pod.(*v1.Pod), nil
}

func (t *targetSelector) selectSingle(ctx context.Context, options Options, log log.Logger, selectFn func(ctx context.Context, options Options, log log.Logger) (bool, interface{}, error)) (interface{}, error) {
	if options.Wait == nil || *options.Wait == true {
		timeout := time.Minute * 10
		if options.Timeout > 0 {
			timeout = time.Duration(options.Timeout) * time.Second
		}

		var out interface{}
		err := wait.PollImmediate(time.Second, timeout, func() (done bool, err error) {
			done, o, err := selectFn(ctx, options, log)
			if err != nil {
				return false, err
			} else if !done {
				return false, nil
			}

			out = o
			return true, nil
		})
		if err != nil {
			if err == wait.ErrWaitTimeout {
				return nil, &NotFoundErr{
					Timeout:  true,
					Selector: options.Selector.String(),
				}
			}

			return nil, err
		}

		return out, nil
	}

	// we try to select a pod
	done, out, err := selectFn(ctx, options, log)
	if err != nil {
		return nil, err
	} else if !done {
		return nil, &NotFoundErr{
			Selector: options.Selector.String(),
		}
	}

	// could be nil
	return out, nil
}

func (t *targetSelector) selectSingleContainer(ctx context.Context, options Options, log log.Logger) (bool, interface{}, error) {
	containers, err := kubectl.NewFilterWithSort(t.client, options.SortPods, options.SortContainers).SelectContainers(ctx, options.Selector)
	if err != nil {
		return false, nil, err
	} else if options.WaitingStrategy != nil {
		return options.WaitingStrategy.SelectContainer(containers, log)
	}

	if len(containers) == 0 {
		return false, nil, nil
	} else if len(containers) == 1 {
		return true, containers[0], nil
	}

	if options.AllowPick {
		names := []string{}
		for _, container := range containers {
			names = append(names, container.Pod.Name+":"+container.Container.Name)
		}

		question := DefaultContainerQuestion
		if options.Question != "" {
			question = options.Question
		}

		containerName, err := log.Question(&survey.QuestionOptions{
			Question: question,
			Options:  names,
		})
		if err != nil {
			return false, nil, err
		}

		for _, container := range containers {
			if container.Pod.Name+":"+container.Container.Name == containerName {
				return true, container, nil
			}
		}

		return false, nil, nil
	}

	// there are two options now, either take the first pod found or error out
	if options.FailIfMultiple {
		return false, nil, fmt.Errorf("found multiple containers for %s", options.Selector.String())
	}

	return true, containers[0], nil
}

func (t *targetSelector) selectSinglePod(ctx context.Context, options Options, log log.Logger) (bool, interface{}, error) {
	pods, err := kubectl.NewFilterWithSort(t.client, options.SortPods, options.SortContainers).SelectPods(ctx, options.Selector)
	if err != nil {
		return false, nil, err
	} else if options.WaitingStrategy != nil {
		return options.WaitingStrategy.SelectPod(pods, log)
	}

	if len(pods) == 0 {
		return false, nil, nil
	} else if len(pods) == 1 {
		return true, pods[0], nil
	}

	if options.AllowPick {
		podNames := []string{}
		for _, pod := range pods {
			podNames = append(podNames, pod.Name)
		}

		question := DefaultPodQuestion
		if options.Question != "" {
			question = options.Question
		}

		podName, err := log.Question(&survey.QuestionOptions{
			Question: question,
			Options:  podNames,
		})
		if err != nil {
			return false, nil, err
		}

		for _, pod := range pods {
			if pod.Name == podName {
				return true, pod, nil
			}
		}

		return false, nil, nil
	}

	// there are two options now, either take the first pod found or error out
	if options.FailIfMultiple {
		return false, nil, fmt.Errorf("found multiple pods for %s", options.Selector.String())
	}

	return true, pods[0], nil
}

func ImageSelectorFromConfig(configImageName string, config *latest.Config, generated *generated.CacheConfig) []string {
	var imageSelector []string
	if configImageName != "" && generated != nil && config != nil {
		imageConfigCache := generated.GetImageCache(configImageName)
		if imageConfigCache.ImageName != "" {
			imageSelector = []string{imageConfigCache.ImageName + ":" + imageConfigCache.Tag}
		} else if config.Images[configImageName] != nil {
			if len(config.Images[configImageName].Tags) > 0 {
				imageSelector = []string{config.Images[configImageName].Image + ":" + config.Images[configImageName].Tags[0]}
			} else {
				imageSelector = []string{config.Images[configImageName].Image}
			}
		}
	}

	return imageSelector
}

type NotFoundErr struct {
	Timeout  bool
	Selector string
}

func (n *NotFoundErr) Error() string {
	if n.Timeout {
		return "timeout: couldn't find a pod / container in time with " + n.Selector
	}

	return "couldn't find a pod / container with " + n.Selector
}
