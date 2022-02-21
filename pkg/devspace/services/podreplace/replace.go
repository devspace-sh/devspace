package podreplace

import (
	"encoding/json"
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"k8s.io/apimachinery/pkg/api/resource"
	"strconv"
	"strings"
	"time"

	yaml2 "github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	ParentKindAnnotation        = "devspace.sh/parent-kind"
	ParentNameAnnotation        = "devspace.sh/parent-name"
	ParentHashAnnotation        = "devspace.sh/parent-hash"
	ReplaceConfigHashAnnotation = "devspace.sh/config-hash"

	ReplicasAnnotation = "devspace.sh/replicas"

	ReplicaSetLabel = "devspace.sh/replaced"
)

type PodReplacer interface {
	// ReplacePod will try to replace a pod with the given config
	ReplacePod(ctx *devspacecontext.Context, devPod *latest.DevPod) error

	// RevertReplacePod will try to revert a pod replacement with the given config
	RevertReplacePod(ctx *devspacecontext.Context, devPod *latest.DevPod) (*selector.SelectedPodContainer, error)
}

func NewPodReplacer() PodReplacer {
	return &replacer{}
}

type replacer struct{}

func (p *replacer) ReplacePod(ctx *devspacecontext.Context, replacePod *latest.DevPod) error {
	// check if there is a replaced pod in the target namespace
	ctx.Log.Info("Try to find replaced pod...")

	// try to find a single patched pod
	selectedPod, err := findSingleReplacedPod(ctx, replacePod)
	if err != nil {
		return errors.Wrap(err, "find patched pod")
	} else if selectedPod != nil {
		shouldUpdate, err := updateNeeded(ctx, selectedPod, replacePod)
		if err != nil {
			return err
		} else if !shouldUpdate {
			ctx.Log.Infof("Found replaced pod %s/%s", selectedPod.Pod.Namespace, selectedPod.Pod.Name)
			return nil
		}
	} else {
		// try to find a single patchable object
		parent, err := p.findScaledDownParentBySelector(ctx, replacePod)
		if err != nil {
			return err
		} else if parent != nil {
			err = deleteLeftOverReplicaSets(ctx, replacePod, parent)
			if err != nil {
				return err
			}

			accessor, _ := meta.Accessor(parent)
			typeAccessor, _ := meta.TypeAccessor(parent)
			ctx.Log.Infof("Reset %s %s/%s", typeAccessor.GetKind(), accessor.GetNamespace(), accessor.GetName())
			err = scaleUpParent(ctx, parent)
			if err != nil {
				return err
			}
		}
	}

	ctx.Log.Info("Try to find replaceable pod...")
	container, parent, err := findSingleReplaceablePodParent(ctx, replacePod)
	if err != nil {
		return err
	}

	// replace the pod
	ctx.Log.Info(fmt.Sprintf("Replacing Pod %s/%s...", container.Pod.Namespace, container.Pod.Name))
	err = replace(ctx, container, parent, replacePod)
	if err != nil {
		return err
	}

	ctx.Log.Donef("Successfully replaced pod %s/%s", container.Pod.Namespace, container.Pod.Name)
	return nil
}

func deleteLeftOverReplicaSets(ctx *devspacecontext.Context, replacePod *latest.DevPod, parent runtime.Object) error {
	accessor, _ := meta.Accessor(parent)
	typeAccessor, _ := meta.TypeAccessor(parent)

	parentName := accessor.GetName()
	parentKind := typeAccessor.GetKind()

	namespace := ctx.KubeClient.Namespace()
	if replacePod.Namespace != "" {
		namespace = replacePod.Namespace
	}

	replicaSets, err := ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(namespace).List(ctx.Context, metav1.ListOptions{LabelSelector: ReplicaSetLabel + "=true"})
	if err != nil {
		return err
	}

	for _, rs := range replicaSets.Items {
		if rs.DeletionTimestamp == nil && rs.Annotations != nil && rs.Annotations[ParentNameAnnotation] == parentName && rs.Annotations[ParentKindAnnotation] == parentKind {
			ctx.Log.Infof("Delete replaced replica set %s/%s", rs.Namespace, rs.Name)
			err = ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(rs.Namespace).Delete(ctx.Context, rs.Name, metav1.DeleteOptions{})
			if err != nil {
				return errors.Wrap(err, "delete pod")
			}
		}
	}

	return nil
}

func updateNeeded(ctx *devspacecontext.Context, pod *selector.SelectedPodContainer, replacePod *latest.DevPod) (bool, error) {
	if pod.Pod.Annotations == nil || pod.Pod.Annotations[ParentKindAnnotation] == "" || pod.Pod.Annotations[ParentNameAnnotation] == "" {
		return true, deleteAndWait(ctx, pod.Pod)
	}

	parent, err := getParentFromReplaced(ctx, pod.Pod.ObjectMeta)
	if err != nil {
		ctx.Log.Infof("Error getting Parent of replaced Pod %s/%s: %v", pod.Pod.Namespace, pod.Pod.Name, err)
		return true, deleteAndWait(ctx, pod.Pod)
	}

	parentHash, err := hashParentPodSpec(ctx, parent, replacePod)
	if err != nil {
		return false, errors.Wrap(err, "hash parent")
	}

	configHash, err := hashConfig(replacePod)
	if err != nil {
		return false, errors.Wrap(err, "hash config")
	}

	// don't update if pod spec & config hash are the same
	if parentHash == pod.Pod.Annotations[ParentHashAnnotation] && configHash == pod.Pod.Annotations[ReplaceConfigHashAnnotation] {
		// make sure parent is downscaled
		err = scaleDownParent(ctx, parent)
		if err != nil {
			ctx.Log.Warnf("Error scaling down parent: %v", err)
		}

		return false, nil
	}

	// delete replaced pod
	ctx.Log.Info("Change detected for replaced Pod " + pod.Pod.Namespace + "/" + pod.Pod.Name)
	err = deleteAndWait(ctx, pod.Pod)
	if err != nil {
		return false, errors.Wrap(err, "delete replaced pod")
	}

	// scale up parent
	ctx.Log.Info("Scaling up parent of replaced pod...")
	err = scaleUpParent(ctx, parent)
	if err != nil {
		return false, err
	}

	return true, nil
}

func getParentFromReplaced(ctx *devspacecontext.Context, obj metav1.ObjectMeta) (runtime.Object, error) {
	var (
		err    error
		parent runtime.Object
	)
	switch obj.GetAnnotations()[ParentKindAnnotation] {
	case "ReplicaSet":
		parent, err = ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(obj.Namespace).Get(ctx.Context, obj.Annotations[ParentNameAnnotation], metav1.GetOptions{})
	case "Deployment":
		parent, err = ctx.KubeClient.KubeClient().AppsV1().Deployments(obj.Namespace).Get(ctx.Context, obj.Annotations[ParentNameAnnotation], metav1.GetOptions{})
	case "StatefulSet":
		parent, err = ctx.KubeClient.KubeClient().AppsV1().StatefulSets(obj.Namespace).Get(ctx.Context, obj.Annotations[ParentNameAnnotation], metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("unrecognized parent kind")
	}

	typeAccessor, _ := meta.TypeAccessor(parent)
	typeAccessor.SetKind(obj.Annotations[ParentKindAnnotation])
	return parent, err
}

func scaleUpParent(ctx *devspacecontext.Context, parent runtime.Object) error {
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
	patch := MergeFrom(clonedParent)
	bytes, err := patch.Data(parent)
	if err != nil {
		return errors.Wrap(err, "create parent patch")
	}

	// patch parent
	switch t := parent.(type) {
	case *appsv1.ReplicaSet:
		_, err = ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(t.Namespace).Patch(ctx.Context, t.Name, patch.Type(), bytes, metav1.PatchOptions{})
	case *appsv1.Deployment:
		_, err = ctx.KubeClient.KubeClient().AppsV1().Deployments(t.Namespace).Patch(ctx.Context, t.Name, patch.Type(), bytes, metav1.PatchOptions{})
	case *appsv1.StatefulSet:
		_, err = ctx.KubeClient.KubeClient().AppsV1().StatefulSets(t.Namespace).Patch(ctx.Context, t.Name, patch.Type(), bytes, metav1.PatchOptions{})
	}
	if err != nil {
		return errors.Wrap(err, "patch parent")
	}

	return nil
}

func deleteAndWait(ctx *devspacecontext.Context, pod *corev1.Pod) error {
	// delete the owning replica set or pod
	if pod.DeletionTimestamp == nil {
		owner := metav1.GetControllerOf(pod)
		if owner != nil && owner.Kind == "ReplicaSet" {
			err := ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(pod.Namespace).Delete(ctx.Context, owner.Name, metav1.DeleteOptions{})
			if err != nil {
				return errors.Wrap(err, "delete replica set")
			}
		} else {
			err := ctx.KubeClient.KubeClient().CoreV1().Pods(pod.Namespace).Delete(ctx.Context, pod.Name, metav1.DeleteOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return nil
				}

				return err
			}
		}
	}

	ctx.Log.Infof("Waiting for replaced pod " + pod.Namespace + "/" + pod.Name + " to get terminated...")
	err := wait.Poll(time.Second, time.Minute*2, func() (bool, error) {
		_, err := ctx.KubeClient.KubeClient().CoreV1().Pods(pod.Namespace).Get(ctx.Context, pod.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		return false, nil
	})
	if err != nil {
		return err
	}

	ctx.Log.StopWait()
	ctx.Log.Donef("Deleted replaced pod %s/%s", pod.Namespace, pod.Name)
	return nil
}

func replace(ctx *devspacecontext.Context, pod *selector.SelectedPodContainer, parent runtime.Object, replacePod *latest.DevPod) error {
	parentHash, err := hashParentPodSpec(ctx, parent, replacePod)
	if err != nil {
		return errors.Wrap(err, "hash parent pod spec")
	}

	configHash, err := hashConfig(replacePod)
	if err != nil {
		return errors.Wrap(err, "hash config")
	}

	copiedPod := pod.Pod.DeepCopyObject().(*corev1.Pod)
	if _, ok := parent.(*appsv1.StatefulSet); ok {
		copiedPod.Spec.Hostname = strings.Replace(pod.Pod.Name, ".", "-", -1)
	}

	// replace the image name
	if replacePod.ReplaceImage != "" {
		err := replaceImageInPodSpec(ctx, &copiedPod.Spec, replacePod)
		if err != nil {
			return err
		}
	}

	// apply the patches
	copiedPod, err = applyPodPatches(copiedPod, replacePod)
	if err != nil {
		return errors.Wrap(err, "apply pod patches")
	}

	// replace paths
	if len(replacePod.PersistPaths) > 0 {
		err := persistPaths(pod.Pod.Name, replacePod, copiedPod)
		if err != nil {
			return err
		}
	}

	// reset the metadata
	copiedPod.ObjectMeta = metav1.ObjectMeta{
		Name:        copiedPod.Name,
		Namespace:   copiedPod.Namespace,
		Labels:      copiedPod.Labels,
		Annotations: copiedPod.Annotations,
	}
	if copiedPod.Annotations == nil {
		copiedPod.Annotations = map[string]string{}
	}
	if copiedPod.Labels == nil {
		copiedPod.Labels = map[string]string{}
	}

	// make sure the pod-template-hash label is deleted
	delete(copiedPod.Labels, "pod-template-hash")
	delete(copiedPod.Labels, "controller-revision-hash")
	delete(copiedPod.Labels, "statefulset.kubernetes.io/pod-name")

	copiedPod.Labels[selector.ReplacedLabel] = "true"
	imageSelector, err := getImageSelector(ctx, replacePod)
	if err != nil {
		return err
	} else if imageSelector != "" {
		copiedPod.Labels[selector.ImageSelectorLabel] = imageSelector
	}
	copiedPod.Annotations[selector.MatchedContainerAnnotation] = pod.Container.Name
	copiedPod.Annotations[ParentHashAnnotation] = parentHash
	copiedPod.Annotations[ReplaceConfigHashAnnotation] = configHash
	copiedPod.Spec.NodeName = ""

	// get pod spec from object
	switch t := parent.(type) {
	case *appsv1.ReplicaSet:
		copiedPod.Annotations[ParentNameAnnotation] = t.Name
		copiedPod.Annotations[ParentKindAnnotation] = "ReplicaSet"
	case *appsv1.Deployment:
		copiedPod.Annotations[ParentNameAnnotation] = t.Name
		copiedPod.Annotations[ParentKindAnnotation] = "Deployment"
	case *appsv1.StatefulSet:
		copiedPod.Annotations[ParentNameAnnotation] = t.Name
		copiedPod.Annotations[ParentKindAnnotation] = "StatefulSet"
	default:
		return fmt.Errorf("unrecognized object")
	}

	// scale down parent
	err = scaleDownParent(ctx, parent)
	if err != nil {
		return errors.Wrap(err, "scale down parent")
	}
	ctx.Log.Donef("Scaled down %s %s/%s", copiedPod.Annotations[ParentKindAnnotation], copiedPod.Namespace, copiedPod.Annotations[ParentNameAnnotation])

	// wait until pod is in terminating mode
	ctx.Log.Info("Waiting for Pod " + pod.Pod.Name + " to get terminated...")
	err = wait.Poll(time.Second*2, time.Minute*2, func() (bool, error) {
		pod, err := ctx.KubeClient.KubeClient().CoreV1().Pods(pod.Pod.Namespace).Get(ctx.Context, pod.Pod.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		// for non stateful set its enough if the pod is still terminating
		if selector.IsPodTerminating(pod) && copiedPod.Annotations[ParentKindAnnotation] != "StatefulSet" {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "wait for original pod to terminate")
	}

	// create a replica set
	replicaSet, err := ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(copiedPod.Namespace).Create(ctx.Context, &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      copiedPod.Name,
			Namespace: copiedPod.Namespace,
			Annotations: map[string]string{
				ParentKindAnnotation: copiedPod.Annotations[ParentKindAnnotation],
				ParentNameAnnotation: copiedPod.Annotations[ParentNameAnnotation],
			},
			Labels: map[string]string{
				ReplicaSetLabel: "true",
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: copiedPod.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      copiedPod.Labels,
					Annotations: copiedPod.Annotations,
				},
				Spec: copiedPod.Spec,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "create copied pod")
	}

	// create a pvc if needed
	if len(replacePod.PersistPaths) > 0 {
		err = createPVC(ctx, copiedPod, replicaSet, replacePod)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				// delete the old one and wait
				_ = ctx.KubeClient.KubeClient().CoreV1().PersistentVolumeClaims(copiedPod.Namespace).Delete(ctx.Context, copiedPod.Name, metav1.DeleteOptions{})
				ctx.Log.Infof("Waiting for old persistent volume claim to terminate")
				err = wait.Poll(time.Second, time.Minute*2, func() (done bool, err error) {
					_, err = ctx.KubeClient.KubeClient().CoreV1().PersistentVolumeClaims(copiedPod.Namespace).Get(ctx.Context, copiedPod.Name, metav1.GetOptions{})
					return kerrors.IsNotFound(err), nil
				})
				if err != nil {
					return errors.Wrap(err, "waiting for pvc to terminate")
				}

				// create the new one
				err = createPVC(ctx, copiedPod, replicaSet, replacePod)
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

func createPVC(ctx *devspacecontext.Context, copiedPod *corev1.Pod, replicaSet *appsv1.ReplicaSet, replacePod *latest.DevPod) error {
	var err error
	size := resource.MustParse("10Gi")
	if replacePod.PersistenceOptions != nil && replacePod.PersistenceOptions.Size != "" {
		size, err = resource.ParseQuantity(replacePod.PersistenceOptions.Size)
		if err != nil {
			return fmt.Errorf("error parsing persistent volume size %s: %v", replacePod.PersistenceOptions.Size, err)
		}
	}

	var storageClassName *string
	if replacePod.PersistenceOptions != nil && replacePod.PersistenceOptions.StorageClassName != "" {
		storageClassName = &replacePod.PersistenceOptions.StorageClassName
	}

	accessModes := []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	if replacePod.PersistenceOptions != nil && replacePod.PersistenceOptions.AccessModes != nil {
		accessModes = []corev1.PersistentVolumeAccessMode{}
		for _, accessMode := range replacePod.PersistenceOptions.AccessModes {
			accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(accessMode))
		}
	}

	name := copiedPod.Name
	if replacePod.PersistenceOptions != nil && replacePod.PersistenceOptions.Name != "" {
		name = replacePod.PersistenceOptions.Name
	}

	_, err = ctx.KubeClient.KubeClient().CoreV1().PersistentVolumeClaims(copiedPod.Namespace).Create(ctx.Context, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: copiedPod.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: appsv1.SchemeGroupVersion.String(),
					Kind:       "ReplicaSet",
					Name:       replicaSet.Name,
					UID:        replicaSet.UID,
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
		if kerrors.IsAlreadyExists(err) && replacePod.PersistenceOptions != nil && replacePod.PersistenceOptions.Name != "" {
			ctx.Log.Infof("PVC %s already exists for replaced pod %s", name, copiedPod.Name)
			return nil
		}

		return err
	}

	ctx.Log.Donef("Created PVC %s for replaced pod %s", name, copiedPod.Name)
	return nil
}

func applyPodPatches(pod *corev1.Pod, replacePod *latest.DevPod) (*corev1.Pod, error) {
	if len(replacePod.Patches) == 0 {
		return pod.DeepCopy(), nil
	}

	raw, err := loader.ApplyPatchesOnObject(convertToInterface(pod), replacePod.Patches)
	if err != nil {
		return nil, err
	}

	// convert back
	rawJSON := convertFromInterface(raw)
	retPod := &corev1.Pod{}
	err = json.Unmarshal(rawJSON, retPod)
	if err != nil {
		return nil, err
	}

	return retPod, nil
}

func hashConfig(replacePod *latest.DevPod) (string, error) {
	out, err := yaml.Marshal(replacePod)
	if err != nil {
		return "", err
	}

	return hash.String(string(out)), nil
}

func hashParentPodSpec(ctx *devspacecontext.Context, obj runtime.Object, replacePod *latest.DevPod) (string, error) {
	cloned := obj.DeepCopyObject()
	var podSpec *corev1.PodTemplateSpec

	// get pod spec from object
	switch t := cloned.(type) {
	case *appsv1.ReplicaSet:
		podSpec = &t.Spec.Template
	case *appsv1.Deployment:
		podSpec = &t.Spec.Template
	case *appsv1.StatefulSet:
		podSpec = &t.Spec.Template
	default:
		return "", fmt.Errorf("unrecognized object")
	}

	// replace the image name
	if replacePod.ReplaceImage != "" {
		err := replaceImageInPodSpec(ctx, &podSpec.Spec, replacePod)
		if err != nil {
			return "", err
		}
	}

	out, err := json.Marshal(podSpec)
	if err != nil {
		return "", err
	}

	return hash.String(string(out)), nil
}

func replaceImageInPodSpec(ctx *devspacecontext.Context, podSpec *corev1.PodSpec, replacePod *latest.DevPod) error {
	imageStr, err := runtimevar.NewRuntimeResolver(true).FillRuntimeVariablesAsString(replacePod.ReplaceImage, ctx.Config, ctx.Dependencies)
	if err != nil {
		return err
	}

	// either replace by labelSelector & containerName
	// or by resolved image name
	if replacePod.LabelSelector != nil {
		if len(podSpec.Containers) > 1 && replacePod.Container == "" {
			return fmt.Errorf("pod spec has more than 1 containers and containerName is an empty string")
		} else if len(podSpec.Containers) == 0 {
			return fmt.Errorf("no containers in pod spec")
		}

		// exchange image name
		for i := range podSpec.Containers {
			if len(podSpec.Containers) == 1 {
				podSpec.Containers[i].Image = imageStr
				break
			} else if podSpec.Containers[i].Name == replacePod.Container {
				podSpec.Containers[i].Image = imageStr
				break
			}
		}
	} else if replacePod.ImageSelector != "" {
		var imageSelector *imageselector.ImageSelector
		if replacePod.ImageSelector != "" {
			imageSelector, err = runtimevar.NewRuntimeResolver(true).FillRuntimeVariablesAsImageSelector(replacePod.ImageSelector, ctx.Config, ctx.Dependencies)
			if err != nil {
				return err
			}
		}

		// exchange image name
		for i := range podSpec.Containers {
			if len(podSpec.Containers) == 1 {
				podSpec.Containers[i].Image = imageStr
				break
			} else if imageSelector != nil && imageselector.CompareImageNames(imageSelector.Image, podSpec.Containers[i].Image) {
				podSpec.Containers[i].Image = imageStr
				break
			}
		}
	}

	return nil
}

func scaleDownParent(ctx *devspacecontext.Context, obj runtime.Object) error {
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
		patch := MergeFrom(cloned)
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
		patch := MergeFrom(cloned)
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
		patch := MergeFrom(cloned)
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

func convertFromInterface(inter map[interface{}]interface{}) []byte {
	out, err := yaml.Marshal(inter)
	if err != nil {
		panic(err)
	}

	retOut, err := yaml2.YAMLToJSON(out)
	if err != nil {
		panic(err)
	}

	return retOut
}

func convertToInterface(str runtime.Object) map[interface{}]interface{} {
	out, err := json.Marshal(str)
	if err != nil {
		panic(err)
	}

	ret := map[interface{}]interface{}{}
	err = yaml.Unmarshal(out, ret)
	if err != nil {
		panic(err)
	}

	return ret
}

func getImageSelector(ctx *devspacecontext.Context, replacePod *latest.DevPod) (string, error) {
	if replacePod.ImageSelector != "" {
		imageSelector, err := runtimevar.NewRuntimeResolver(true).FillRuntimeVariablesAsImageSelector(replacePod.ImageSelector, ctx.Config, ctx.Dependencies)
		if err != nil {
			return "", err
		} else if imageSelector == nil {
			return "", fmt.Errorf("couldn't resolve image selector: %v", replacePod.ImageSelector)
		}

		return hash.String(imageSelector.Image)[:32], nil
	}

	return "", nil
}

func findSingleReplacedPod(ctx *devspacecontext.Context, replacePod *latest.DevPod) (*selector.SelectedPodContainer, error) {
	replicaSet, err := findReplacedPodReplicaSet(ctx, replacePod)
	if err != nil {
		return nil, err
	} else if replicaSet == nil {
		return nil, nil
	}

	labelSelector := map[string]string{
		selector.ReplacedLabel: "true",
	}
	imageSelector, err := getImageSelector(ctx, replacePod)
	if err != nil {
		return nil, err
	} else if imageSelector != "" {
		labelSelector[selector.ImageSelectorLabel] = imageSelector
	} else if len(replacePod.LabelSelector) > 0 {
		for k, v := range replacePod.LabelSelector {
			labelSelector[k] = v
		}
	} else {
		return nil, fmt.Errorf("imageName, imageSelector or labelSelector need to be defined")
	}

	// create selector
	targetOptions := targetselector.NewEmptyOptions().
		ApplyConfigParameter(replacePod.Container, labelSelector, nil, replacePod.Namespace, "").
		WithTimeout(30).
		WithWaitingStrategy(targetselector.NewUntilNotTerminatingStrategy(0)).
		WithSkipInitContainers(true)

	// get container
	selected, err := targetselector.NewTargetSelector(targetOptions).SelectSingleContainer(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return nil, err
	}

	return selected, nil
}

func findReplacedPodReplicaSet(ctx *devspacecontext.Context, replacePod *latest.DevPod) (runtime.Object, error) {
	namespace := ctx.KubeClient.Namespace()
	if replacePod.Namespace != "" {
		namespace = replacePod.Namespace
	}

	// replicaSets
	replicaSets, err := ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(namespace).List(ctx.Context, metav1.ListOptions{LabelSelector: ReplicaSetLabel + "=true"})
	if err != nil {
		return nil, errors.Wrap(err, "list ReplicaSets")
	}
	for _, replicaSet := range replicaSets.Items {
		parent, err := getParentFromReplaced(ctx, replicaSet.ObjectMeta)
		if err != nil {
			if kerrors.IsNotFound(err) {
				// delete replica set
				err = ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(namespace).Delete(ctx.Context, replicaSet.Name, metav1.DeleteOptions{})
				if err != nil {
					ctx.Log.Info("Error deleting replica set %s/%s: %v", namespace, replicaSet.Name, err)
				}

				continue
			}

			return nil, err
		}

		// get pod spec from object
		var (
			annotations map[string]string
			podSpec     *corev1.PodTemplateSpec
		)
		switch t := parent.(type) {
		case *appsv1.ReplicaSet:
			annotations = t.Annotations
			podSpec = &t.Spec.Template
		case *appsv1.Deployment:
			annotations = t.Annotations
			podSpec = &t.Spec.Template
		case *appsv1.StatefulSet:
			annotations = t.Annotations
			podSpec = &t.Spec.Template
		}

		matched, err := matchesSelector(ctx, annotations, podSpec, replacePod)
		if err != nil {
			return nil, err
		} else if matched {
			replicaSet.Kind = "ReplicaSet"
			return &replicaSet, nil
		}
	}

	return nil, nil
}

func findSingleReplaceablePodParent(ctx *devspacecontext.Context, replacePod *latest.DevPod) (*selector.SelectedPodContainer, runtime.Object, error) {
	var (
		err           error
		imageSelector []string
	)
	if replacePod.ImageSelector != "" {
		imageSelectorObject, err := runtimevar.NewRuntimeResolver(true).FillRuntimeVariablesAsImageSelector(replacePod.ImageSelector, ctx.Config, ctx.Dependencies)
		if err != nil {
			return nil, nil, err
		}

		imageSelector = []string{imageSelectorObject.Image}
	}

	// create selector
	targetOptions := targetselector.NewEmptyOptions().
		ApplyConfigParameter(replacePod.Container, replacePod.LabelSelector, imageSelector, replacePod.Namespace, "").
		WithTimeout(300).
		WithWaitingStrategy(targetselector.NewUntilNotTerminatingStrategy(time.Second * 2)).
		WithSkipInitContainers(true)

	container, err := targetselector.NewTargetSelector(targetOptions).SelectSingleContainer(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return nil, nil, err
	}

	parent, err := getParent(ctx, container.Pod)
	if err != nil {
		return nil, nil, errors.Wrap(err, "get pod parent")
	}

	return container, parent, nil
}

func getParent(ctx *devspacecontext.Context, pod *corev1.Pod) (runtime.Object, error) {
	controller := metav1.GetControllerOf(pod)
	if controller == nil {
		return nil, fmt.Errorf("pod was not created by a ReplicaSet, Deployment or StatefulSet, patching only works if pod was created by one of those resources")
	}

	// replica set / deployment ?
	if controller.Kind == "ReplicaSet" {
		// try to find the replica set, we ignore the group version for now
		replicaSet, err := ctx.KubeClient.KubeClient().AppsV1().ReplicaSets(pod.Namespace).Get(ctx.Context, controller.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, fmt.Errorf("unrecognized owning ReplicaSet %s group version: %s", controller.Name, controller.APIVersion)
			}

			return nil, err
		}

		replicaSetOwner := metav1.GetControllerOf(replicaSet)
		if replicaSetOwner == nil {
			return replicaSet, nil
		}

		// is deployment?
		if replicaSetOwner.Kind == "Deployment" {
			deployment, err := ctx.KubeClient.KubeClient().AppsV1().Deployments(pod.Namespace).Get(ctx.Context, replicaSetOwner.Name, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return nil, fmt.Errorf("unrecognized owning Deployment %s group version: %s", replicaSetOwner.Name, replicaSetOwner.APIVersion)
				}

				return nil, err
			}

			// we stop here, if the Deployment is owned by something else we just ignore it for now
			return deployment, nil
		}

		return nil, fmt.Errorf("unrecognized owner of ReplicaSet %s: %s %s %s", replicaSet.Name, replicaSetOwner.Kind, replicaSetOwner.APIVersion, replicaSetOwner.Name)
	}

	// statefulset?
	if controller.Kind == "StatefulSet" {
		statefulSet, err := ctx.KubeClient.KubeClient().AppsV1().StatefulSets(pod.Namespace).Get(ctx.Context, controller.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, fmt.Errorf("unrecognized owning StatefulSet %s group version: %s", controller.Name, controller.APIVersion)
			}

			return nil, err
		}

		return statefulSet, nil
	}

	return nil, fmt.Errorf("unrecognized owner of Pod %s: %s %s %s", pod.Name, controller.Kind, controller.APIVersion, controller.Name)
}
