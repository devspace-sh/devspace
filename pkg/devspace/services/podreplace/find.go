package podreplace

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

func findTargetByKindName(ctx devspacecontext.Context, kind, namespace, name string) (runtime.Object, error) {
	var (
		err    error
		parent runtime.Object
	)
	switch kind {
	case "ReplicaSet":
		parent, err = ctx.KubeClient().KubeClient().AppsV1().ReplicaSets(namespace).Get(ctx.Context(), name, metav1.GetOptions{})
	case "Deployment":
		parent, err = ctx.KubeClient().KubeClient().AppsV1().Deployments(namespace).Get(ctx.Context(), name, metav1.GetOptions{})
	case "StatefulSet":
		parent, err = ctx.KubeClient().KubeClient().AppsV1().StatefulSets(namespace).Get(ctx.Context(), name, metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("unrecognized parent kind")
	}
	if err != nil {
		return nil, err
	}

	typeAccessor, _ := meta.TypeAccessor(parent)
	typeAccessor.SetAPIVersion("apps/v1")
	typeAccessor.SetKind(kind)
	return parent, nil
}

func findTargetBySelector(ctx devspacecontext.Context, devPod *latest.DevPod, filter func(obj metav1.Object) bool) (runtime.Object, error) {
	namespace := ctx.KubeClient().Namespace()
	if devPod.Namespace != "" {
		namespace = devPod.Namespace
	}

	// deployments
	deployments, err := ctx.KubeClient().KubeClient().AppsV1().Deployments(namespace).List(ctx.Context(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list Deployments")
	}
	for _, d := range deployments.Items {
		if filter != nil && !filter(&d) {
			continue
		}

		matched, err := matchesSelector(ctx, &d.Spec.Template, devPod)
		if err != nil {
			return nil, err
		} else if matched {
			d.Kind = "Deployment"
			return &d, nil
		}
	}

	// replicaSets
	replicaSets, err := ctx.KubeClient().KubeClient().AppsV1().ReplicaSets(namespace).List(ctx.Context(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list ReplicaSets")
	}
	for _, d := range replicaSets.Items {
		if len(d.OwnerReferences) > 0 || (filter != nil && !filter(&d)) {
			continue
		}

		matched, err := matchesSelector(ctx, &d.Spec.Template, devPod)
		if err != nil {
			return nil, err
		} else if matched {
			d.Kind = "ReplicaSet"
			return &d, nil
		}
	}

	// statefulSets
	statefulSets, err := ctx.KubeClient().KubeClient().AppsV1().StatefulSets(namespace).List(ctx.Context(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list StatefulSets")
	}
	for _, d := range statefulSets.Items {
		if filter != nil && !filter(&d) {
			continue
		}

		matched, err := matchesSelector(ctx, &d.Spec.Template, devPod)
		if err != nil {
			return nil, err
		} else if matched {
			d.Kind = "StatefulSet"
			return &d, nil
		}
	}

	return nil, nil
}

func matchesSelector(ctx devspacecontext.Context, pod *corev1.PodTemplateSpec, devPod *latest.DevPod) (bool, error) {
	if len(devPod.LabelSelector) > 0 {
		labelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: devPod.LabelSelector,
		})
		if err != nil {
			return false, err
		}

		return labelSelector.Matches(labels.Set(pod.Labels)), nil
	} else if devPod.ImageSelector != "" {
		containers, err := matchesImageSelector(ctx, pod, devPod)
		if err != nil {
			return false, err
		}

		return len(containers) > 0, nil
	}

	return false, nil
}

func matchesImageSelector(ctx devspacecontext.Context, pod *corev1.PodTemplateSpec, devPod *latest.DevPod) ([]string, error) {
	var matchingContainers []string
	if devPod.ImageSelector != "" {
		imageSelector, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir(), true).FillRuntimeVariablesAsImageSelector(ctx.Context(), devPod.ImageSelector, ctx.Config(), ctx.Dependencies())
		if err != nil {
			return nil, err
		}

		loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
			for i := range pod.Spec.Containers {
				if devContainer.Container != "" && pod.Spec.Containers[i].Name != devContainer.Container {
					continue
				}

				if imageselector.CompareImageNames(imageSelector.Image, pod.Spec.Containers[i].Image) {
					matchingContainers = append(matchingContainers, pod.Spec.Containers[i].Name)
					break
				}
			}
			return true
		})

		return matchingContainers, nil
	}

	return matchingContainers, nil
}
