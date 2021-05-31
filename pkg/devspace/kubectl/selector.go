package kubectl

import (
	"context"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"strings"
)

const (
	MatchedContainerAnnotation = "devspace.sh/container"
	ImageNameLabel             = "devspace.sh/imageName"
	ImageSelectorLabel         = "devspace.sh/imageSelector"

	ReplacedLabel = "devspace.sh/replaced"
)

var SortPodsByNewest = func(pods []*k8sv1.Pod, i, j int) bool {
	return pods[i].CreationTimestamp.Unix() > pods[j].CreationTimestamp.Unix()
}

var SortContainersByNewest = func(pods []*SelectedPodContainer, i, j int) bool {
	return pods[i].Pod.CreationTimestamp.Unix() > pods[j].Pod.CreationTimestamp.Unix()
}

var FilterNonRunningPods = func(p *k8sv1.Pod) bool {
	return GetPodStatus(p) != "Running"
}

var FilterNonRunningContainers = func(p *k8sv1.Pod, c *k8sv1.Container) bool {
	if p.DeletionTimestamp != nil {
		return true
	}
	for _, cs := range p.Status.InitContainerStatuses {
		if cs.Name == c.Name && cs.State.Running != nil {
			return false
		}
	}
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Name == c.Name && cs.State.Running != nil {
			return false
		}
	}
	return true
}

type SelectedPodContainer struct {
	Pod       *k8sv1.Pod
	Container *k8sv1.Container
}

type Selector struct {
	ImageSelector      []imageselector.ImageSelector
	LabelSelector      string
	Pod                string
	ContainerName      string
	Namespace          string
	SkipInitContainers bool

	FilterPod       FilterPod
	FilterContainer FilterContainer
}

func (s Selector) String() string {
	if len(s.ImageSelector) == 0 && len(s.LabelSelector) == 0 && s.Pod == "" {
		return "everything selector"
	}

	strs := []string{}
	if len(s.ImageSelector) > 0 {
		sa := []string{}
		for _, c := range s.ImageSelector {
			sa = append(sa, c.ConfigImageName+"="+c.Image)
		}

		strs = append(strs, "image selector: "+strings.Join(sa, ","))
	}
	if len(s.LabelSelector) > 0 {
		strs = append(strs, "label selector: "+s.LabelSelector)
	}
	if s.Pod != "" {
		strs = append(strs, "pod name: "+s.Pod)
	}

	return strings.Join(strs, ", ")
}

type FilterPod func(p *k8sv1.Pod) bool
type FilterContainer func(p *k8sv1.Pod, c *k8sv1.Container) bool

type SortPods func(pods []*k8sv1.Pod, i, j int) bool
type SortContainers func(containers []*SelectedPodContainer, i, j int) bool

type Filter interface {
	SelectContainers(ctx context.Context, selectors ...Selector) ([]*SelectedPodContainer, error)
	SelectPods(ctx context.Context, selectors ...Selector) ([]*k8sv1.Pod, error)
}

type filter struct {
	client Client

	sortPods       SortPods
	sortContainers SortContainers
}

func NewFilter(client Client) Filter {
	return &filter{
		client: client,
	}
}

func NewFilterWithSort(client Client, sortPods SortPods, sortContainers SortContainers) Filter {
	return &filter{
		client:         client,
		sortPods:       sortPods,
		sortContainers: sortContainers,
	}
}

func (f *filter) SelectPods(ctx context.Context, selectors ...Selector) ([]*k8sv1.Pod, error) {
	retList, err := f.SelectContainers(ctx, selectors...)
	if err != nil {
		return nil, err
	}

	pods := podsFromPodContainer(retList)
	if f.sortPods != nil {
		sort.Slice(pods, func(i, j int) bool {
			return f.sortPods(pods, i, j)
		})
	}

	return pods, nil
}

func (f *filter) SelectContainers(ctx context.Context, selectors ...Selector) ([]*SelectedPodContainer, error) {
	retList := []*SelectedPodContainer{}
	for _, s := range selectors {
		namespace := f.client.Namespace()
		if s.Namespace != "" {
			namespace = s.Namespace
		}

		if s.LabelSelector != "" || (len(s.ImageSelector) == 0 && s.Pod == "") {
			containersByLabelSelector, err := byLabelSelector(ctx, f.client, namespace, s.LabelSelector, s.ContainerName, s.FilterPod, s.FilterContainer, s.SkipInitContainers)
			if err != nil {
				return nil, errors.Wrap(err, "pods by label selector")
			}

			retList = append(retList, containersByLabelSelector...)
		}

		containersByImage, err := byImageName(ctx, f.client, namespace, s.ImageSelector, s.FilterPod, s.FilterContainer, s.SkipInitContainers)
		if err != nil {
			return nil, errors.Wrap(err, "pods by image name")
		}

		containersByName, err := byPodName(ctx, f.client, namespace, s.Pod, s.ContainerName, s.FilterPod, s.FilterContainer, s.SkipInitContainers)
		if err != nil {
			return nil, errors.Wrap(err, "pods by label selector")
		}

		retList = append(retList, containersByImage...)
		retList = append(retList, containersByName...)
	}

	retList = deduplicate(retList)
	if f.sortContainers != nil {
		sort.Slice(retList, func(i, j int) bool {
			return f.sortContainers(retList, i, j)
		})
	}

	return retList, nil
}

func deduplicate(stack []*SelectedPodContainer) []*SelectedPodContainer {
	retStack := []*SelectedPodContainer{}
	for _, s := range stack {
		if !contains(retStack, key(s.Pod.Namespace, s.Pod.Name, s.Container.Name)) {
			retStack = append(retStack, s)
		}
	}
	return retStack
}

func podsFromPodContainer(stack []*SelectedPodContainer) []*k8sv1.Pod {
	retPods := []*k8sv1.Pod{}
	for _, s := range stack {
		if !containsPod(retPods, key(s.Pod.Namespace, s.Pod.Name, "")) {
			retPods = append(retPods, s.Pod)
		}
	}
	return retPods
}

func containsPod(stack []*k8sv1.Pod, k string) bool {
	for _, s := range stack {
		if key(s.Namespace, s.Name, "") == k {
			return true
		}
	}
	return false
}

func contains(stack []*SelectedPodContainer, k string) bool {
	for _, s := range stack {
		if key(s.Pod.Namespace, s.Pod.Name, s.Container.Name) == k {
			return true
		}
	}
	return false
}

func key(namespace string, pod string, container string) string {
	return namespace + "/" + pod + "/" + container
}

func byPodName(ctx context.Context, client Client, namespace string, name string, containerName string, skipPod FilterPod, skipContainer FilterContainer, skipInit bool) ([]*SelectedPodContainer, error) {
	if name == "" {
		return nil, nil
	}

	retPods := []*SelectedPodContainer{}
	pod, err := client.KubeClient().CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return retPods, nil
		}

		return nil, errors.Wrap(err, "get pod")
	}
	if skipPod != nil && skipPod(pod) {
		return nil, nil
	}

	if !skipInit {
		for _, container := range pod.Spec.InitContainers {
			if skipContainer != nil && skipContainer(pod, &container) {
				continue
			}
			if containerName != "" && container.Name != containerName {
				continue
			}

			retContainer := container
			retPods = append(retPods, &SelectedPodContainer{
				Pod:       pod,
				Container: &retContainer,
			})
		}
	}
	for _, container := range pod.Spec.Containers {
		if skipContainer != nil && skipContainer(pod, &container) {
			continue
		}
		if containerName != "" && container.Name != containerName {
			continue
		}

		retContainer := container
		retPods = append(retPods, &SelectedPodContainer{
			Pod:       pod,
			Container: &retContainer,
		})
	}

	return retPods, nil
}

func byLabelSelector(ctx context.Context, client Client, namespace string, labelSelector string, containerName string, skipPod FilterPod, skipContainer FilterContainer, skipInit bool) ([]*SelectedPodContainer, error) {
	retPods := []*SelectedPodContainer{}
	podList, err := client.KubeClient().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, errors.Wrap(err, "list pods")
	}

	for _, pod := range podList.Items {
		if skipPod != nil && skipPod(&pod) {
			continue
		}

		if !skipInit {
			for _, container := range pod.Spec.InitContainers {
				if skipContainer != nil && skipContainer(&pod, &container) {
					continue
				}
				if containerName != "" && container.Name != containerName {
					continue
				}

				retPod := pod
				retContainer := container
				retPods = append(retPods, &SelectedPodContainer{
					Pod:       &retPod,
					Container: &retContainer,
				})
			}
		}
		for _, container := range pod.Spec.Containers {
			if skipContainer != nil && skipContainer(&pod, &container) {
				continue
			}
			if containerName != "" && container.Name != containerName {
				continue
			}

			retPod := pod
			retContainer := container
			retPods = append(retPods, &SelectedPodContainer{
				Pod:       &retPod,
				Container: &retContainer,
			})
		}
	}
	return retPods, nil
}

func byImageName(ctx context.Context, client Client, namespace string, imageSelector []imageselector.ImageSelector, skipPod FilterPod, skipContainer FilterContainer, skipInit bool) ([]*SelectedPodContainer, error) {
	retPods := []*SelectedPodContainer{}
	if len(imageSelector) > 0 {
		podList, err := client.KubeClient().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "list pods")
		}

		for _, pod := range podList.Items {
			if skipPod != nil && skipPod(&pod) {
				continue
			}

			if !skipInit {
				for _, container := range pod.Spec.InitContainers {
					for _, imageName := range imageSelector {
						if skipContainer != nil && skipContainer(&pod, &container) {
							continue
						}

						if imageselector.CompareImageNames(imageName, container.Image) {
							retPod := pod
							retContainer := container
							retPods = append(retPods, &SelectedPodContainer{
								Pod:       &retPod,
								Container: &retContainer,
							})
						}
					}
				}
			}
			for _, container := range pod.Spec.Containers {
				for _, imageName := range imageSelector {
					if skipContainer != nil && skipContainer(&pod, &container) {
						continue
					}

					// check if it is a replaced pod and if yes, check if the imageName and container name matches
					if pod.Labels != nil && pod.Labels[ReplacedLabel] == "true" && pod.Annotations != nil && pod.Annotations[MatchedContainerAnnotation] == container.Name {
						if pod.Labels[ImageNameLabel] != "" && pod.Labels[ImageNameLabel] == imageName.ConfigImageName {
							retPod := pod
							retContainer := container
							retPods = append(retPods, &SelectedPodContainer{
								Pod:       &retPod,
								Container: &retContainer,
							})
						} else if pod.Labels[ImageSelectorLabel] != "" && pod.Labels[ImageSelectorLabel] == hash.String(imageName.ImageSelector)[:32] {
							retPod := pod
							retContainer := container
							retPods = append(retPods, &SelectedPodContainer{
								Pod:       &retPod,
								Container: &retContainer,
							})
						}
						continue
					}

					if imageselector.CompareImageNames(imageName, container.Image) {
						retPod := pod
						retContainer := container
						retPods = append(retPods, &SelectedPodContainer{
							Pod:       &retPod,
							Container: &retContainer,
						})
					}
				}
			}
		}
	}
	return retPods, nil
}
