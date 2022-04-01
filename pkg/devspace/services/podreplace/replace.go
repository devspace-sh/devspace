package podreplace

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	patch2 "github.com/loft-sh/devspace/pkg/util/patch"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	"strconv"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	TargetKindAnnotation       = "devspace.sh/parent-kind"
	TargetNameAnnotation       = "devspace.sh/parent-name"
	DevPodConfigHashAnnotation = "devspace.sh/config-hash"

	ReplicasAnnotation = "devspace.sh/replicas"
)

type PodReplacer interface {
	// ReplacePod will try to replace a pod with the given config
	ReplacePod(ctx *devspacecontext.Context, devPod *latest.DevPod) error

	// RevertReplacePod will try to revert a pod replacement with the given config
	RevertReplacePod(ctx *devspacecontext.Context, devPodCache *remotecache.DevPodCache) (bool, error)
}

func NewPodReplacer() PodReplacer {
	return &replacer{}
}

type replacer struct{}

func (p *replacer) ReplacePod(ctx *devspacecontext.Context, devPod *latest.DevPod) error {
	namespace := devPod.Namespace
	if namespace == "" {
		namespace = ctx.KubeClient.Namespace()
	}

	devPodCache, ok := ctx.Config.RemoteCache().GetDevPod(devPod.Name)
	if !ok {
		devPodCache.Name = devPod.Name
		devPodCache.Namespace = namespace
	}

	// did we already replace a pod?
	if devPodCache.Deployment != "" {
		// check if there is a replaced pod in the target namespace
		ctx.Log.Debug("Try to find replaced deployment...")

		// find the replaced replica set
		deployment, err := ctx.KubeClient.KubeClient().AppsV1().Deployments(devPodCache.Namespace).Get(ctx.Context, devPodCache.Deployment, metav1.GetOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return errors.Wrap(err, "find devspace deployment")
			}

			// fallthrough to recreate replicaSet
		} else {
			recreateNeeded, err := updateNeeded(ctx, deployment, devPod)
			if err != nil {
				return err
			} else if !recreateNeeded {
				return nil
			}

			// fallthrough to recreate replicaSet
		}
	}

	// try to find a replaceable deployment statefulset etc.
	target, err := findTargetBySelector(ctx, devPod, nil)
	if err != nil {
		return err
	} else if target == nil {
		return fmt.Errorf("couldn't find a matching deployment, statefulset or replica set")
	}

	// make sure we already save the cache here
	devPodCache.TargetKind = target.GetObjectKind().GroupVersionKind().Kind
	devPodCache.TargetName = target.(metav1.Object).GetName()
	devPodCache.Deployment = target.(metav1.Object).GetName() + "-devspace"
	ctx.Config.RemoteCache().SetDevPod(devPodCache.Name, devPodCache)
	err = ctx.Config.RemoteCache().Save(ctx.Context, ctx.KubeClient)
	if err != nil {
		return err
	}

	// replace the pod
	ctx.Log.Debugf("Replacing %s %s...", devPodCache.TargetKind, devPodCache.TargetName)
	err = p.replace(ctx, devPodCache.Deployment, target, devPod)
	if err != nil {
		return err
	}

	ctx.Log.Debugf("Successfully replaced %s %s", devPodCache.TargetKind, devPodCache.TargetName)
	return nil
}

func updateNeeded(ctx *devspacecontext.Context, deployment *appsv1.Deployment, devPod *latest.DevPod) (recreateNeeded bool, err error) {
	if deployment.Annotations == nil || deployment.Annotations[TargetKindAnnotation] == "" || deployment.Annotations[TargetNameAnnotation] == "" {
		return true, deleteDeployment(ctx, deployment)
	}

	target, err := findTargetByKindName(ctx, deployment.Annotations[TargetKindAnnotation], deployment.Namespace, deployment.Annotations[TargetNameAnnotation])
	if err != nil {
		if kerrors.IsNotFound(err) {
			return true, deleteDeployment(ctx, deployment)
		}

		ctx.Log.Debugf("error getting target for deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
		return false, err
	}

	newDeployment, err := buildDeployment(ctx, deployment.Name, target, devPod)
	if err != nil {
		return false, err
	}

	configHash, err := hashConfig(devPod)
	if err != nil {
		return false, errors.Wrap(err, "hash config")
	}

	err = scaleDownTarget(ctx, target)
	if err != nil {
		ctx.Log.Warnf("Error scaling down target: %v", err)
	}

	// don't update if pod spec & config hash are the same
	if apiequality.Semantic.DeepEqual(newDeployment.Spec.Template, deployment.Spec.Template) && configHash == deployment.Annotations[DevPodConfigHashAnnotation] {
		// make sure target is downscaled
		ctx.Log.Debugf("No changes required in replaced deployment %s", deployment.Name)
		return false, nil
	}

	// update deployment´
	originalDeployment := deployment.DeepCopy()
	deployment.Spec.Template = newDeployment.Spec.Template
	deployment.Annotations[DevPodConfigHashAnnotation] = configHash
	patch := patch2.MergeFrom(originalDeployment)
	patchBytes, err := patch.Data(deployment)
	if err != nil {
		return false, err
	} else if string(patchBytes) == "{}" {
		ctx.Log.Debugf("No changes required in replaced deployment %s", deployment.Name)
		return false, nil
	}

	ctx.Log.Debugf("Update replaced deployment with patch:\n %s", string(patchBytes))

	_, err = ctx.KubeClient.KubeClient().AppsV1().Deployments(deployment.Namespace).Patch(ctx.Context, deployment.Name, patch.Type(), patchBytes, metav1.PatchOptions{})
	if err != nil {
		if kerrors.IsInvalid(err) {
			ctx.Log.Debugf("Recreate deployment because it is invalid: %v", err)
			return true, deleteDeployment(ctx, deployment)
		}

		return false, err
	}

	return false, nil
}

func deleteDeployment(ctx *devspacecontext.Context, deployment *appsv1.Deployment) error {
	// delete the owning deployment or pod
	err := ctx.KubeClient.KubeClient().AppsV1().Deployments(deployment.Namespace).Delete(ctx.Context, deployment.Name, metav1.DeleteOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return errors.Wrap(err, "delete deployment")
		}
	}

	return nil
}

func (p *replacer) replace(ctx *devspacecontext.Context, deploymentName string, target runtime.Object, devPod *latest.DevPod) error {
	deploymentObj, err := buildDeployment(ctx, deploymentName, target, devPod)
	if err != nil {
		return err
	}

	// scale down parent
	err = scaleDownTarget(ctx, target)
	if err != nil {
		return errors.Wrap(err, "scale down target")
	}
	ctx.Log.Debugf("Scaled down %s %s", deploymentObj.Annotations[TargetKindAnnotation], deploymentObj.Annotations[TargetNameAnnotation])

	// create the deployment
	deployment, err := ctx.KubeClient.KubeClient().AppsV1().Deployments(deploymentObj.Namespace).Create(ctx.Context, deploymentObj, metav1.CreateOptions{})
	if err != nil {
		if kerrors.IsAlreadyExists(err) {
			ctx.Log.Info("Pod was already replaced, retrying to update the configuration")
			return p.ReplacePod(ctx, devPod)
		}

		return errors.Wrap(err, "create deployment")
	}

	// create a pvc if needed
	hasPersistPath := false
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		if len(devContainer.PersistPaths) > 0 {
			hasPersistPath = true
			return false
		}
		return true
	})
	if hasPersistPath {
		err = createPVC(ctx, deployment, devPod)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				// delete the old one and wait
				_ = ctx.KubeClient.KubeClient().CoreV1().PersistentVolumeClaims(deployment.Namespace).Delete(ctx.Context, deployment.Name, metav1.DeleteOptions{})
				ctx.Log.Infof("Waiting for old persistent volume claim to terminate")
				err = wait.Poll(time.Second, time.Minute*2, func() (done bool, err error) {
					_, err = ctx.KubeClient.KubeClient().CoreV1().PersistentVolumeClaims(deployment.Namespace).Get(ctx.Context, deployment.Name, metav1.GetOptions{})
					return kerrors.IsNotFound(err), nil
				})
				if err != nil {
					return errors.Wrap(err, "waiting for pvc to terminate")
				}

				// create the new one
				err = createPVC(ctx, deployment, devPod)
				if err != nil {
					return errors.Wrap(err, "create persistent volume claim")
				}
			} else {
				return errors.Wrap(err, "create persistent volume claim")
			}
		}
	}

	return nil
}

func createPVC(ctx *devspacecontext.Context, deployment *appsv1.Deployment, devPod *latest.DevPod) error {
	var err error
	size := resource.MustParse("10Gi")
	if devPod.PersistenceOptions != nil && devPod.PersistenceOptions.Size != "" {
		size, err = resource.ParseQuantity(devPod.PersistenceOptions.Size)
		if err != nil {
			return fmt.Errorf("error parsing persistent volume size %s: %v", devPod.PersistenceOptions.Size, err)
		}
	}

	var storageClassName *string
	if devPod.PersistenceOptions != nil && devPod.PersistenceOptions.StorageClassName != "" {
		storageClassName = &devPod.PersistenceOptions.StorageClassName
	}

	accessModes := []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	if devPod.PersistenceOptions != nil && devPod.PersistenceOptions.AccessModes != nil {
		accessModes = []corev1.PersistentVolumeAccessMode{}
		for _, accessMode := range devPod.PersistenceOptions.AccessModes {
			accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(accessMode))
		}
	}

	name := deployment.Name
	if devPod.PersistenceOptions != nil && devPod.PersistenceOptions.Name != "" {
		name = devPod.PersistenceOptions.Name
	}

	_, err = ctx.KubeClient.KubeClient().CoreV1().PersistentVolumeClaims(deployment.Namespace).Create(ctx.Context, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: deployment.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: appsv1.SchemeGroupVersion.String(),
					Kind:       "Deployment",
					Name:       deployment.Name,
					UID:        deployment.UID,
					Controller: ptr.Bool(true),
				},
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: size,
				},
			},
			StorageClassName: storageClassName,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		if kerrors.IsAlreadyExists(err) && devPod.PersistenceOptions != nil && devPod.PersistenceOptions.Name != "" {
			ctx.Log.Infof("PVC %s already exists for replaced pod %s", name, deployment.Name)
			return nil
		}

		return err
	}

	ctx.Log.Donef("Created PVC %s for replaced pod %s", name, deployment.Name)
	return nil
}

func scaleDownTarget(ctx *devspacecontext.Context, obj runtime.Object) error {
	cloned := obj.DeepCopyObject()

	// update object based on type
	switch t := obj.(type) {
	case *appsv1.ReplicaSet:
		if t.Annotations == nil {
			t.Annotations = map[string]string{}
		}

		replicas := 1
		if t.Spec.Replicas != nil {
			replicas = int(*t.Spec.Replicas)
		}

		if replicas == 0 {
			return nil
		}

		t.Annotations[ReplicasAnnotation] = strconv.Itoa(replicas)
		t.Spec.Replicas = ptr.Int32(0)
		patch := patch2.MergeFrom(cloned)
		bytes, err := patch.Data(t)
		if err != nil {
			return err
		}

		_, err = ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(t.Namespace).Patch(ctx.Context, t.Name, patch.Type(), bytes, metav1.PatchOptions{})
		if err != nil {
			return err
		}

		return nil
	case *appsv1.Deployment:
		if t.Annotations == nil {
			t.Annotations = map[string]string{}
		}

		replicas := 1
		if t.Spec.Replicas != nil {
			replicas = int(*t.Spec.Replicas)
		}

		if replicas == 0 {
			return nil
		}

		t.Annotations[ReplicasAnnotation] = strconv.Itoa(replicas)
		t.Spec.Replicas = ptr.Int32(0)
		patch := patch2.MergeFrom(cloned)
		bytes, err := patch.Data(t)
		if err != nil {
			return err
		}

		_, err = ctx.KubeClient.KubeClient().AppsV1().Deployments(t.Namespace).Patch(ctx.Context, t.Name, patch.Type(), bytes, metav1.PatchOptions{})
		if err != nil {
			return err
		}

		return nil
	case *appsv1.StatefulSet:
		if t.Annotations == nil {
			t.Annotations = map[string]string{}
		}

		replicas := 1
		if t.Spec.Replicas != nil {
			replicas = int(*t.Spec.Replicas)
		}

		if replicas == 0 {
			return nil
		}

		t.Annotations[ReplicasAnnotation] = strconv.Itoa(replicas)
		t.Spec.Replicas = ptr.Int32(0)
		patch := patch2.MergeFrom(cloned)
		bytes, err := patch.Data(t)
		if err != nil {
			return err
		}

		_, err = ctx.KubeClient.KubeClient().AppsV1().StatefulSets(t.Namespace).Patch(ctx.Context, t.Name, patch.Type(), bytes, metav1.PatchOptions{})
		if err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("unrecognized object")
}
