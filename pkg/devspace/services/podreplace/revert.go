package podreplace

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dependencytypes "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/util"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

func (p *replacer) RevertReplacePod(ctx context.Context, client kubectl.Client, config config.Config, dependencies []dependencytypes.Dependency, replacePod *latest.ReplacePod, log log.Logger) (*selector.SelectedPodContainer, error) {
	// check if there is a replaced pod in the target namespace
	log.Info("Try to find replaced pod...")

	// try to find a single patched pod
	selectedPod, err := findSingleReplacedPod(ctx, client, replacePod, config, dependencies, log)
	if err != nil {
		return nil, errors.Wrap(err, "find patched pod")
	} else if selectedPod == nil {
		parent, err := p.findScaledDownParentBySelector(ctx, client, config, dependencies, replacePod)
		if err != nil {
			return nil, err
		} else if parent == nil {
			return nil, nil
		}

		err = deleteLeftOverReplicaSets(ctx, client, replacePod, parent, log)
		if err != nil {
			return nil, err
		}

		accessor, _ := meta.Accessor(parent)
		typeAccessor, _ := meta.TypeAccessor(parent)
		log.Infof("Scale up %s %s/%s", typeAccessor.GetKind(), accessor.GetNamespace(), accessor.GetName())
		return nil, scaleUpParent(ctx, client, parent)
	}

	if selectedPod.Pod.Annotations == nil || selectedPod.Pod.Annotations[ParentKindAnnotation] == "" || selectedPod.Pod.Annotations[ParentNameAnnotation] == "" {
		return selectedPod, deleteAndWait(ctx, client, selectedPod.Pod, log)
	}

	parent, err := getParentFromReplaced(ctx, client, selectedPod.Pod.ObjectMeta)
	if err != nil {
		// log.Infof("Error getting Parent of replaced Pod %s/%s: %v", selectedPod.Pod.Namespace, selectedPod.Pod.Name, err)
		return selectedPod, deleteAndWait(ctx, client, selectedPod.Pod, log)
	}

	// delete replaced pods
	err = deleteLeftOverReplicaSets(ctx, client, replacePod, parent, log)
	if err != nil {
		return nil, err
	}

	// scale up parent
	log.Info("Scaling up parent of replaced pod...")
	err = scaleUpParent(ctx, client, parent)
	if err != nil {
		return nil, err
	}

	return selectedPod, nil
}

func (p *replacer) findScaledDownParentBySelector(ctx context.Context, client kubectl.Client, config config.Config, dependencies []dependencytypes.Dependency, replacePod *latest.ReplacePod) (runtime.Object, error) {
	namespace := client.Namespace()
	if replacePod.Namespace != "" {
		namespace = replacePod.Namespace
	}

	// deployments
	deployments, err := client.KubeClient().AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list Deployments")
	}
	for _, d := range deployments.Items {
		matched, err := matchesSelector(d.Annotations, &d.Spec.Template, config, dependencies, replacePod)
		if err != nil {
			return nil, err
		} else if matched {
			d.Kind = "Deployment"
			return &d, nil
		}
	}

	// replicaSets
	replicaSets, err := client.KubeClient().AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list ReplicaSets")
	}
	for _, d := range replicaSets.Items {
		if len(d.OwnerReferences) > 0 {
			continue
		}

		matched, err := matchesSelector(d.Annotations, &d.Spec.Template, config, dependencies, replacePod)
		if err != nil {
			return nil, err
		} else if matched {
			d.Kind = "ReplicaSet"
			return &d, nil
		}
	}

	// statefulSets
	statefulSets, err := client.KubeClient().AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list StatefulSets")
	}
	for _, d := range statefulSets.Items {
		matched, err := matchesSelector(d.Annotations, &d.Spec.Template, config, dependencies, replacePod)
		if err != nil {
			return nil, err
		} else if matched {
			d.Kind = "StatefulSet"
			return &d, nil
		}
	}

	return nil, nil
}

func matchesSelector(annotations map[string]string, pod *corev1.PodTemplateSpec, config config.Config, dependencies []dependencytypes.Dependency, replacePod *latest.ReplacePod) (bool, error) {
	if annotations == nil || annotations[ReplicasAnnotation] == "" {
		return false, nil
	}

	if len(replacePod.LabelSelector) > 0 {
		labelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: replacePod.LabelSelector,
		})
		if err != nil {
			return false, err
		}

		return labelSelector.Matches(labels.Set(pod.Labels)), nil
	} else if replacePod.ImageSelector != "" {
		imageSelector, err := util.ResolveImageAsImageSelector(replacePod.ImageSelector, config, dependencies)
		if err != nil {
			return false, err
		}

		// compare image
		for i := range pod.Spec.Containers {
			if imageselector.CompareImageNames(*imageSelector, pod.Spec.Containers[i].Image) {
				return true, nil
			}
		}

		return false, nil
	}

	return false, nil
}
