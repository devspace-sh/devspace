package podreplace

import (
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

func (p *replacer) RevertReplacePod(ctx *devspacecontext.Context, devPodCache *remotecache.DevPodCache) (bool, error) {
	// check if there is a replaced pod in the target namespace
	ctx.Log.Info("Try to find replaced pod...")

	namespace := devPodCache.Namespace
	if namespace == "" {
		namespace = ctx.KubeClient.Namespace()
	}

	// delete replica set & scale up parent
	deleted := false
	if devPodCache.ReplicaSet != "" {
		err := ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(namespace).Delete(ctx.Context, devPodCache.ReplicaSet, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return false, errors.Wrap(err, "delete devspace replica set")
			}
		} else {
			deleted = true
		}
	}

	// scale up parent
	parent, err := getParentByKindName(ctx, devPodCache.ParentKind, namespace, devPodCache.ParentName)
	if err != nil {
		ctx.Log.Debugf("Error getting parent by name: %v", err)
		ctx.Config.RemoteCache().DeleteDevPod(devPodCache.Name)
		return deleted, nil
	}

	// scale up parent
	ctx.Log.Info("Scaling up parent of replaced pod...")
	err = scaleUpParent(ctx, parent)
	if err != nil {
		return false, err
	}

	ctx.Config.RemoteCache().DeleteDevPod(devPodCache.Name)
	return deleted, ctx.Config.RemoteCache().Save(ctx.Context, ctx.KubeClient)
}

func (p *replacer) findScaledDownParentBySelector(ctx *devspacecontext.Context, replacePod *latest.DevPod) (runtime.Object, error) {
	namespace := ctx.KubeClient.Namespace()
	if replacePod.Namespace != "" {
		namespace = replacePod.Namespace
	}

	// deployments
	deployments, err := ctx.KubeClient.KubeClient().AppsV1().Deployments(namespace).List(ctx.Context, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list Deployments")
	}
	for _, d := range deployments.Items {
		matched, err := matchesSelector(ctx, d.Annotations, &d.Spec.Template, replacePod)
		if err != nil {
			return nil, err
		} else if matched {
			d.Kind = "Deployment"
			return &d, nil
		}
	}

	// replicaSets
	replicaSets, err := ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(namespace).List(ctx.Context, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list ReplicaSets")
	}
	for _, d := range replicaSets.Items {
		if len(d.OwnerReferences) > 0 {
			continue
		}

		matched, err := matchesSelector(ctx, d.Annotations, &d.Spec.Template, replacePod)
		if err != nil {
			return nil, err
		} else if matched {
			d.Kind = "ReplicaSet"
			return &d, nil
		}
	}

	// statefulSets
	statefulSets, err := ctx.KubeClient.KubeClient().AppsV1().StatefulSets(namespace).List(ctx.Context, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list StatefulSets")
	}
	for _, d := range statefulSets.Items {
		matched, err := matchesSelector(ctx, d.Annotations, &d.Spec.Template, replacePod)
		if err != nil {
			return nil, err
		} else if matched {
			d.Kind = "StatefulSet"
			return &d, nil
		}
	}

	return nil, nil
}

func matchesSelector(ctx *devspacecontext.Context, annotations map[string]string, pod *corev1.PodTemplateSpec, replacePod *latest.DevPod) (bool, error) {
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
		imageSelector, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir, true).FillRuntimeVariablesAsImageSelector(replacePod.ImageSelector, ctx.Config, ctx.Dependencies)
		if err != nil {
			return false, err
		}

		// compare image
		for i := range pod.Spec.Containers {
			if imageselector.CompareImageNames(imageSelector.Image, pod.Spec.Containers[i].Image) {
				return true, nil
			}
		}

		return false, nil
	}

	return false, nil
}
