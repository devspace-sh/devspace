package targetselector

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"

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
	selector selector.Selector

	allowPick bool
	question  string

	wait    *bool
	timeout int64

	failIfMultiple bool
	sortContainers selector.SortContainers

	waitingStrategy WaitingStrategy
}

func NewEmptyOptions() Options {
	return Options{
		selector: selector.Selector{
			FilterContainer: selector.FilterTerminatingContainers,
		},
		sortContainers: selector.SortContainersByNewest,
	}
}

func NewOptionsFromFlags(containerName string, labelSelector string, imageSelector []string, namespace string, pod string) Options {
	return Options{
		selector: selector.Selector{
			ImageSelector:   imageSelector,
			LabelSelector:   labelSelector,
			Pod:             pod,
			ContainerName:   containerName,
			Namespace:       namespace,
			FilterContainer: selector.FilterNonRunningContainers,
		},
		sortContainers: selector.SortContainersByNewest,
	}
}

func (o Options) WithPod(pod string) Options {
	newOptions := o
	newOptions.selector.Pod = pod
	return newOptions
}

func (o Options) WithLabelSelector(labelSelector string) Options {
	newOptions := o
	newOptions.selector.LabelSelector = labelSelector
	return newOptions
}

func (o Options) WithContainer(container string) Options {
	newOptions := o
	newOptions.selector.ContainerName = container
	return newOptions
}

func (o Options) WithQuestion(question string) Options {
	newOptions := o
	newOptions.question = question
	return newOptions
}

func (o Options) WithImageSelector(imageSelector []string) Options {
	newOptions := o
	newOptions.selector.ImageSelector = imageSelector
	return newOptions
}

func (o Options) WithWait(wait bool) Options {
	newOptions := o
	newOptions.wait = &wait
	return newOptions
}

func (o Options) WithTimeout(timeout int64) Options {
	newOptions := o
	newOptions.timeout = timeout
	return newOptions
}

func (o Options) WithNamespace(namespace string) Options {
	newOptions := o
	newOptions.selector.Namespace = namespace
	return newOptions
}

func (o Options) WithSkipInitContainers(skip bool) Options {
	newOptions := o
	newOptions.selector.SkipInitContainers = skip
	return newOptions
}

func (o Options) WithContainerFilter(containerFilter selector.FilterContainer) Options {
	newOptions := o
	newOptions.selector.FilterContainer = containerFilter
	return newOptions
}

func (o Options) WithWaitingStrategy(waitingStrategy WaitingStrategy) Options {
	newOptions := o
	newOptions.waitingStrategy = waitingStrategy
	return newOptions
}

func (o Options) WithPick(allowPick bool) Options {
	newOptions := o
	newOptions.allowPick = allowPick
	return newOptions
}

func (o Options) ApplyConfigParameter(containerName string, labelSelector map[string]string, imageSelector []string, namespace string, pod string) Options {
	newOptions := o
	if containerName != "" && o.selector.ContainerName == "" {
		newOptions.selector.ContainerName = containerName
	}
	if labelSelector != nil && o.selector.LabelSelector == "" {
		newOptions.selector.LabelSelector = labels.Set(labelSelector).String()
	}
	if namespace != "" && o.selector.Namespace == "" {
		newOptions.selector.Namespace = namespace
	}
	if pod != "" && o.selector.Pod == "" {
		newOptions.selector.Pod = pod
	}
	if len(imageSelector) > 0 && len(o.selector.ImageSelector) == 0 {
		newOptions.selector.ImageSelector = imageSelector
	}
	return newOptions
}

func ToStringImageSelector(imageSelector []imageselector.ImageSelector) []string {
	imageSelectors := []string{}
	for _, i := range imageSelector {
		if i.Image == "" {
			continue
		}

		imageSelectors = append(imageSelectors, i.Image)
	}

	return imageSelectors
}

type TargetSelector interface {
	SelectSinglePod(ctx context.Context, client kubectl.Client, log log.Logger) (*v1.Pod, error)
	SelectSingleContainer(ctx context.Context, client kubectl.Client, log log.Logger) (*selector.SelectedPodContainer, error)

	WithContainer(container string) TargetSelector
}

// targetSelector is the struct that will select a target
type targetSelector struct {
	options Options
}

// NewTargetSelector creates a new target selector for selecting a target pod or container
func NewTargetSelector(options Options) TargetSelector {
	return &targetSelector{
		options: options,
	}
}

func (t *targetSelector) WithContainer(container string) TargetSelector {
	return &targetSelector{
		options: t.options.WithContainer(container),
	}
}

func (t *targetSelector) SelectSingleContainer(ctx context.Context, client kubectl.Client, log log.Logger) (*selector.SelectedPodContainer, error) {
	log.Debugf("Start selecting a single container with selector %v", t.options.selector.String())

	if t.options.waitingStrategy != nil {
		t.options.waitingStrategy = t.options.waitingStrategy.Reset()
	}

	container, err := t.selectSingle(ctx, client, t.options, log, t.selectSingleContainer)
	if err != nil {
		return nil, err
	} else if container == nil {
		return nil, nil
	}

	return container.(*selector.SelectedPodContainer), nil
}

func (t *targetSelector) SelectSinglePod(ctx context.Context, client kubectl.Client, log log.Logger) (*v1.Pod, error) {
	log.Debugf("Start selecting a single pod with selector %v", t.options.selector.String())

	if t.options.waitingStrategy != nil {
		t.options.waitingStrategy = t.options.waitingStrategy.Reset()
	}

	pod, err := t.selectSingle(ctx, client, t.options, log, t.selectSinglePod)
	if err != nil {
		return nil, err
	} else if pod == nil {
		return nil, nil
	}

	return pod.(*v1.Pod), nil
}

func (t *targetSelector) selectSingle(ctx context.Context, client kubectl.Client, options Options, log log.Logger, selectFn func(ctx context.Context, client kubectl.Client, options Options, log log.Logger) (bool, interface{}, error)) (interface{}, error) {
	if options.wait == nil || *options.wait {
		timeout := time.Minute * 10
		if options.timeout > 0 {
			timeout = time.Duration(options.timeout) * time.Second
		}

		var out interface{}
		err := wait.PollImmediateWithContext(ctx, time.Millisecond*500, timeout, func(ctx context.Context) (done bool, err error) {
			done, o, err := selectFn(ctx, client, options, log)
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
					Selector: options.selector.String(),
				}
			}

			return nil, err
		}

		return out, nil
	}

	// we try to select a pod
	done, out, err := selectFn(ctx, client, options, log)
	if err != nil {
		return nil, err
	} else if !done {
		return nil, &NotFoundErr{
			Selector: options.selector.String(),
		}
	}

	// could be nil
	return out, nil
}

func (t *targetSelector) selectSingleContainer(ctx context.Context, client kubectl.Client, options Options, log log.Logger) (bool, interface{}, error) {
	containers, err := selector.NewFilterWithSort(client, options.sortContainers).SelectContainers(ctx, options.selector)
	if err != nil {
		return false, nil, err
	} else if options.waitingStrategy != nil {
		return options.waitingStrategy.SelectContainer(ctx, client, options.selector.Namespace, containers, log)
	}

	if len(containers) == 0 {
		return false, nil, nil
	} else if len(containers) == 1 {
		return true, containers[0], nil
	}

	if options.allowPick {
		names := []string{}
		for _, container := range containers {
			names = append(names, container.Pod.Name+":"+container.Container.Name)
		}

		question := DefaultContainerQuestion
		if options.question != "" {
			question = options.question
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
	if options.failIfMultiple {
		return false, nil, fmt.Errorf("found multiple containers for %s", options.selector.String())
	}

	return true, containers[0], nil
}

func (t *targetSelector) selectSinglePod(ctx context.Context, client kubectl.Client, options Options, log log.Logger) (bool, interface{}, error) {
	stack, err := selector.NewFilterWithSort(client, options.sortContainers).SelectContainers(ctx, options.selector)
	if err != nil {
		return false, nil, err
	}

	// transform stack
	pods := selector.PodsFromPodContainer(stack)
	if options.waitingStrategy != nil {
		namespace := options.selector.Namespace
		if namespace == "" {
			namespace = client.Namespace()
		}

		return options.waitingStrategy.SelectPod(ctx, client, namespace, pods, log)
	}

	if len(pods) == 0 {
		return false, nil, nil
	} else if len(pods) == 1 {
		return true, pods[0], nil
	}

	if options.allowPick {
		podNames := []string{}
		for _, pod := range pods {
			podNames = append(podNames, pod.Name)
		}

		question := DefaultPodQuestion
		if options.question != "" {
			question = options.question
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
	if options.failIfMultiple {
		return false, nil, fmt.Errorf("found multiple pods for %s", options.selector.String())
	}

	return true, pods[0], nil
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
