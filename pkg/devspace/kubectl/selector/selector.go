package selector

import (
	"context"
	"sort"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/imageselector"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	MatchedContainerAnnotation = "devspace.sh/container"
	ImageSelectorAnnotation    = "devspace.sh/imageSelector"

	ReplacedLabel = "devspace.sh/replaced"
)

var SortPodsByNewest = func(pods []*corev1.Pod, i, j int) bool {
	return pods[i].CreationTimestamp.Unix() > pods[j].CreationTimestamp.Unix()
}

var SortContainersByNewest = func(pods []*SelectedPodContainer, i, j int) bool {
	if pods[i].Pod.Name == pods[j].Pod.Name {
		// this is needed for containers with the same image where we want to say that normal containers take prio over init containers
		return initContainerPos(pods[i].Container.Name, pods[i].Pod) < initContainerPos(pods[j].Container.Name, pods[j].Pod)
	}

	return pods[i].Pod.CreationTimestamp.Unix() > pods[j].Pod.CreationTimestamp.Unix()
}

func initContainerPos(container string, pod *corev1.Pod) int {
	for i, c := range pod.Spec.InitContainers {
		if c.Name == container {
			return i
		}
	}
	return -1
}

var FilterTerminatingContainers = func(p *corev1.Pod, c *corev1.Container) bool {
	return IsPodTerminating(p)
}

var FilterNonRunningContainers = func(p *corev1.Pod, c *corev1.Container) bool {
	if IsPodTerminating(p) {
		return true
	}
	for _, cs := range p.Status.InitContainerStatuses {
		if cs.Name == c.Name && cs.Ready && cs.State.Running != nil {
			return false
		}
	}
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Name == c.Name && cs.Ready && cs.State.Running != nil {
			return false
		}
	}
	return true
}

type SelectedPodContainer struct {
	Pod       *corev1.Pod
	Container *corev1.Container
}

type Selector struct {
	ImageSelector      []string `json:"imageSelector"`
	LabelSelector      string   `json:"labelSelector"`
	Pod                string   `json:"pod"`
	ContainerName      string   `json:"containerName"`
	Namespace          string   `json:"namespace"`
	SkipInitContainers bool     `json:"skipInitContainers"`

	FilterContainer FilterContainer `json:"-"`
}

func (s Selector) String() string {
	if len(s.ImageSelector) == 0 && len(s.LabelSelector) == 0 && s.Pod == "" {
		return "everything selector"
	}

	strs := []string{}
	if len(s.ImageSelector) > 0 {
		strs = append(strs, "image selector: "+strings.Join(s.ImageSelector, ","))
	}
	if len(s.LabelSelector) > 0 {
		if s.ContainerName != "" {
			strs = append(strs, "label selector: "+s.LabelSelector+" - container: "+s.ContainerName)
		} else {
			strs = append(strs, "label selector: "+s.LabelSelector)
		}
	}
	if s.Pod != "" {
		strs = append(strs, "pod name: "+s.Pod)
	}

	return strings.Join(strs, ", ")
}

type FilterContainer func(p *corev1.Pod, c *corev1.Container) bool
type SortContainers func(containers []*SelectedPodContainer, i, j int) bool

type Filter interface {
	SelectContainers(ctx context.Context, selectors ...Selector) ([]*SelectedPodContainer, error)
}

type filter struct {
	client kubectl.Client

	sortContainers SortContainers
}

func NewFilter(client kubectl.Client) Filter {
	return &filter{
		client: client,
	}
}

func NewFilterWithSort(client kubectl.Client, sortContainers SortContainers) Filter {
	return &filter{
		client:         client,
		sortContainers: sortContainers,
	}
}

func (f *filter) SelectContainers(ctx context.Context, selectors ...Selector) ([]*SelectedPodContainer, error) {
	retList := []*SelectedPodContainer{}
	for _, s := range selectors {
		namespace := f.client.Namespace()
		if s.Namespace != "" {
			namespace = s.Namespace
		}

		if s.LabelSelector != "" || (len(s.ImageSelector) == 0 && s.Pod == "") {
			containersByLabelSelector, err := byLabelSelector(ctx, f.client, namespace, s.LabelSelector, s.ContainerName, s.FilterContainer, s.SkipInitContainers)
			if err != nil {
				return nil, errors.Wrap(err, "pods by label selector")
			}

			retList = append(retList, containersByLabelSelector...)
		}

		containersByImage, err := byImageName(ctx, f.client, namespace, s.ImageSelector, s.ContainerName, s.FilterContainer, s.SkipInitContainers)
		if err != nil {
			return nil, errors.Wrap(err, "pods by image name")
		}

		containersByName, err := byPodName(ctx, f.client, namespace, s.Pod, s.ContainerName, s.FilterContainer, s.SkipInitContainers)
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

func PodsFromPodContainer(stack []*SelectedPodContainer) []*corev1.Pod {
	retPods := []*corev1.Pod{}
	for _, s := range stack {
		if !containsPod(retPods, key(s.Pod.Namespace, s.Pod.Name, "")) {
			retPods = append(retPods, s.Pod)
		}
	}

	sort.Slice(retPods, func(i, j int) bool {
		return SortPodsByNewest(retPods, i, j)
	})
	return retPods
}

func containsPod(stack []*corev1.Pod, k string) bool {
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

func byPodName(ctx context.Context, client kubectl.Client, namespace string, name string, containerName string, skipContainer FilterContainer, skipInit bool) ([]*SelectedPodContainer, error) {
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

func byLabelSelector(ctx context.Context, client kubectl.Client, namespace string, labelSelector string, containerName string, skipContainer FilterContainer, skipInit bool) ([]*SelectedPodContainer, error) {
	retPods := []*SelectedPodContainer{}
	podList, err := client.KubeClient().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, errors.Wrap(err, "list pods")
	}

	for _, pod := range podList.Items {
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

func byImageName(ctx context.Context, client kubectl.Client, namespace string, imageSelector []string, containerName string, skipContainer FilterContainer, skipInit bool) ([]*SelectedPodContainer, error) {
	retPods := []*SelectedPodContainer{}
	if len(imageSelector) > 0 {
		podList, err := client.KubeClient().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "list pods")
		}

		for _, pod := range podList.Items {
			if !skipInit {
				for _, container := range pod.Spec.InitContainers {
					for _, imageName := range imageSelector {
						if skipContainer != nil && skipContainer(&pod, &container) {
							continue
						}
						if containerName != "" && container.Name != containerName {
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
					if containerName != "" && container.Name != containerName {
						continue
					}

					// check if it is a replaced pod and if yes, check if the imageName and container name matches
					containers := map[string]bool{}
					if pod.Annotations != nil && pod.Annotations[MatchedContainerAnnotation] != "" {
						splitted := strings.Split(pod.Annotations[MatchedContainerAnnotation], ";")
						for _, s := range splitted {
							containers[s] = true
						}
					}
					if pod.Labels != nil && pod.Labels[ReplacedLabel] == "true" && containers[container.Name] {
						if pod.Annotations != nil && pod.Annotations[ImageSelectorAnnotation] != "" && pod.Annotations[ImageSelectorAnnotation] == imageName {
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

func IsPodTerminating(pod *corev1.Pod) bool {
	return pod.DeletionTimestamp != nil || strings.Contains(pod.Status.Reason, "Evicted")
}
