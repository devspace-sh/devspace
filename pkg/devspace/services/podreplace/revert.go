package podreplace

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	patch2 "github.com/loft-sh/devspace/pkg/util/patch"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"strconv"
	"strings"
)

func (p *replacer) RevertReplacePod(ctx devspacecontext.Context, devPodCache *remotecache.DevPodCache, options *deploy.PurgeOptions) (bool, error) {
	if options == nil {
		options = &deploy.PurgeOptions{}
	}

	// check if there is a replaced pod in the target namespace
	ctx.Log().Debug("Try to find replaced pod...")

	// get root name
	rootName, ok := values.RootNameFrom(ctx.Context())
	if ok && !options.ForcePurge && len(devPodCache.Projects) > 0 && (len(devPodCache.Projects) > 1 || devPodCache.Projects[0] != rootName) {
		newProjects := []string{}
		for _, p := range devPodCache.Projects {
			if p == rootName {
				continue
			}

			newProjects = append(newProjects, p)
		}

		devPodCache.Projects = newProjects
		ctx.Log().Infof("Skip reverting dev %s as it is still in use by other DevSpace project(s) '%s'. Run with '--force-purge' to force reverting", devPodCache.Name, strings.Join(devPodCache.Projects, "', '"))
		ctx.Config().RemoteCache().SetDevPod(devPodCache.Name, *devPodCache)
		return false, ctx.Config().RemoteCache().Save(ctx.Context(), ctx.KubeClient())
	}

	// find correct namespace
	namespace := devPodCache.Namespace
	if namespace == "" {
		namespace = ctx.KubeClient().Namespace()
	}

	// delete replica set & scale up parent
	deleted := false
	if devPodCache.Deployment != "" {
		err := ctx.KubeClient().KubeClient().AppsV1().Deployments(namespace).Delete(ctx.Context(), devPodCache.Deployment, metav1.DeleteOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return false, errors.Wrap(err, "delete devspace deployment")
			}
		} else {
			deleted = true
		}
	}

	// scale up parent
	parent, err := findTargetByKindName(ctx, devPodCache.TargetKind, namespace, devPodCache.TargetName)
	if err != nil {
		ctx.Log().Debugf("Error getting parent by name: %v", err)
		ctx.Config().RemoteCache().DeleteDevPod(devPodCache.Name)
		return deleted, nil
	}

	// scale up parent
	ctx.Log().Infof("Scaling up %s %s...", devPodCache.TargetKind, devPodCache.TargetName)
	err = scaleUpTarget(ctx, parent)
	if err != nil {
		return false, err
	}

	ctx.Config().RemoteCache().DeleteDevPod(devPodCache.Name)
	return deleted, ctx.Config().RemoteCache().Save(ctx.Context(), ctx.KubeClient())
}

func scaleUpTarget(ctx devspacecontext.Context, parent runtime.Object) error {
	clonedParent := parent.DeepCopyObject()
	metaParent, err := meta.Accessor(parent)
	if err != nil {
		return errors.Wrap(err, "parent accessor")
	}

	// check if required annotation is there
	annotations := metaParent.GetAnnotations()
	if annotations == nil || annotations[ReplicasAnnotation] == "" {
		return nil
	}

	// scale up parent
	oldReplica, err := strconv.Atoi(annotations[ReplicasAnnotation])
	if err != nil {
		return errors.Wrap(err, "parse old replicas")
	} else if oldReplica == 0 {
		return nil
	}

	oldReplica32 := int32(oldReplica)
	switch t := parent.(type) {
	case *appsv1.ReplicaSet:
		t.Spec.Replicas = &oldReplica32
	case *appsv1.Deployment:
		t.Spec.Replicas = &oldReplica32
	case *appsv1.StatefulSet:
		t.Spec.Replicas = &oldReplica32
	}

	// delete replicas annotation
	delete(annotations, ReplicasAnnotation)
	metaParent.SetAnnotations(annotations)

	// create patch
	patch := patch2.MergeFrom(clonedParent)
	bytes, err := patch.Data(parent)
	if err != nil {
		return errors.Wrap(err, "create parent patch")
	}

	// patch parent
	switch t := parent.(type) {
	case *appsv1.ReplicaSet:
		_, err = ctx.KubeClient().KubeClient().AppsV1().ReplicaSets(t.Namespace).Patch(ctx.Context(), t.Name, patch.Type(), bytes, metav1.PatchOptions{})
	case *appsv1.Deployment:
		_, err = ctx.KubeClient().KubeClient().AppsV1().Deployments(t.Namespace).Patch(ctx.Context(), t.Name, patch.Type(), bytes, metav1.PatchOptions{})
	case *appsv1.StatefulSet:
		_, err = ctx.KubeClient().KubeClient().AppsV1().StatefulSets(t.Namespace).Patch(ctx.Context(), t.Name, patch.Type(), bytes, metav1.PatchOptions{})
	}
	if err != nil {
		return errors.Wrap(err, "patch parent")
	}

	return nil
}
