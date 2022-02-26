package podreplace

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

func buildReplicaSet(ctx *devspacecontext.Context, name string, target runtime.Object, devPod *latest.DevPod) (*appsv1.ReplicaSet, error) {
	configHash, err := hashConfig(devPod)
	if err != nil {
		return nil, errors.Wrap(err, "hash config")
	}

	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: target.(metav1.Object).GetNamespace(),
			Annotations: map[string]string{
				DevPodConfigHashAnnotation: configHash,
			},
			Labels: map[string]string{
				ReplicaSetLabel: "true",
			},
		},
	}

	podTemplate := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
	}
	switch t := target.(type) {
	case *appsv1.ReplicaSet:
		replicaSet.Annotations[TargetNameAnnotation] = t.Name
		replicaSet.Annotations[TargetKindAnnotation] = "ReplicaSet"
		podTemplate.Spec = *t.Spec.Template.Spec.DeepCopy()
	case *appsv1.Deployment:
		replicaSet.Annotations[TargetNameAnnotation] = t.Name
		replicaSet.Annotations[TargetKindAnnotation] = "Deployment"
		podTemplate.Spec = *t.Spec.Template.Spec.DeepCopy()
	case *appsv1.StatefulSet:
		replicaSet.Annotations[TargetNameAnnotation] = t.Name
		replicaSet.Annotations[TargetKindAnnotation] = "StatefulSet"
		podTemplate.Spec = *t.Spec.Template.Spec.DeepCopy()
		podTemplate.Spec.Hostname = strings.Replace(t.Name+"-0", ".", "-", -1)
		for _, pvc := range t.Spec.VolumeClaimTemplates {
			pvcName := pvc.Name
			if pvcName == "" {
				pvcName = "data-" + t.Name
			}

			podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, corev1.Volume{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			})
		}
	default:
		return nil, fmt.Errorf("unrecognized object")
	}

	// get matching container if imageSelector
	containers, err := matchesImageSelector(ctx, podTemplate, devPod)
	if err != nil {
		return nil, err
	} else if len(containers) > 0 {
		replicaSet.Annotations[selector.MatchedContainerAnnotation] = strings.Join(containers, ";")
	}

	// replace the image names
	err = replaceImagesInPodSpec(ctx, &podTemplate.Spec, devPod)
	if err != nil {
		return nil, err
	}

	// apply the patches
	podTemplate, err = applyPodPatches(podTemplate, devPod)
	if err != nil {
		return nil, errors.Wrap(err, "apply pod patches")
	}

	// check if terminal and modify pod
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		if devContainer.Terminal == nil || devContainer.Terminal.Disabled || devContainer.Terminal.DisableReplace {
			return true
		}

		if devContainer.Container == "" && len(podTemplate.Spec.Containers) > 1 {
			names := []string{}
			for _, c := range podTemplate.Spec.Containers {
				names = append(names, c.Name)
			}

			err = fmt.Errorf("couldn't open terminal as multiple containers were found %s, but no containerName was specified", strings.Join(names, " "))
			return false
		}

		for i, con := range podTemplate.Spec.Containers {
			if devContainer.Container == "" || con.Name == devContainer.Container {
				podTemplate.Spec.Containers[i].ReadinessProbe = nil
				podTemplate.Spec.Containers[i].StartupProbe = nil
				podTemplate.Spec.Containers[i].LivenessProbe = nil
				podTemplate.Spec.Containers[i].Command = []string{"sleep", "1000000000"}
				podTemplate.Spec.Containers[i].Args = []string{}
			}
		}

		return false
	})
	if err != nil {
		return nil, errors.Wrap(err, "apply terminal")
	}

	// replace paths
	if len(devPod.PersistPaths) > 0 {
		err := persistPaths(name, devPod, podTemplate)
		if err != nil {
			return nil, err
		}
	}

	// reset the metadata
	podTemplate.Labels[selector.ReplacedLabel] = "true"
	imageSelector, err := hashImageSelector(ctx, devPod)
	if err != nil {
		return nil, err
	} else if imageSelector != "" {
		podTemplate.Annotations[selector.ImageSelectorAnnotation] = imageSelector
	}

	replicaSet.Spec = appsv1.ReplicaSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: podTemplate.ObjectMeta.Labels,
		},
		Template: *podTemplate,
	}
	return replicaSet, nil
}

func hashConfig(replacePod *latest.DevPod) (string, error) {
	out, err := yaml.Marshal(replacePod)
	if err != nil {
		return "", err
	}

	return hash.String(string(out)), nil
}

func replaceImagesInPodSpec(ctx *devspacecontext.Context, podSpec *corev1.PodSpec, devPod *latest.DevPod) error {
	var err error
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		if devContainer.ReplaceImage == "" {
			return true
		}
		err = replaceImageInPodSpec(ctx, podSpec, devPod.LabelSelector, devPod.ImageSelector, devContainer.Container, devContainer.ReplaceImage)
		if err != nil {
			return false
		}
		return true
	})

	return err
}

func replaceImageInPodSpec(ctx *devspacecontext.Context, podSpec *corev1.PodSpec, labelSelector map[string]string, imageSelector string, container, replaceImage string) error {
	if len(podSpec.Containers) == 0 {
		return fmt.Errorf("no containers in pod spec")
	}

	imageStr, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir, true).FillRuntimeVariablesAsString(replaceImage, ctx.Config, ctx.Dependencies)
	if err != nil {
		return err
	}

	if container != "" {
		for i := range podSpec.Containers {
			if podSpec.Containers[i].Name == container {
				podSpec.Containers[i].Image = imageStr
				break
			}
		}
	} else if labelSelector != nil {
		if len(podSpec.Containers) > 1 {
			return fmt.Errorf("pod spec has more than 1 containers and containerName is an empty string")
		}

		// exchange image name
		if len(podSpec.Containers) == 1 {
			podSpec.Containers[0].Image = imageStr
		}
	} else if imageSelector != "" {
		if len(podSpec.Containers) == 1 {
			podSpec.Containers[0].Image = imageStr
		} else {
			var imageSelectorPtr *imageselector.ImageSelector
			if imageSelector != "" {
				imageSelectorPtr, err = runtimevar.NewRuntimeResolver(ctx.WorkingDir, true).FillRuntimeVariablesAsImageSelector(replaceImage, ctx.Config, ctx.Dependencies)
				if err != nil {
					return err
				}
			}

			// exchange image name
			for i := range podSpec.Containers {
				if imageSelectorPtr != nil && imageselector.CompareImageNames(imageSelectorPtr.Image, podSpec.Containers[i].Image) {
					podSpec.Containers[i].Image = imageStr
					break
				}
			}
		}
	}

	return nil
}

func hashImageSelector(ctx *devspacecontext.Context, replacePod *latest.DevPod) (string, error) {
	if replacePod.ImageSelector != "" {
		imageSelector, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir, true).FillRuntimeVariablesAsImageSelector(replacePod.ImageSelector, ctx.Config, ctx.Dependencies)
		if err != nil {
			return "", err
		} else if imageSelector == nil {
			return "", fmt.Errorf("couldn't resolve image selector: %v", replacePod.ImageSelector)
		}

		return hash.String(imageSelector.Image)[:32], nil
	}

	return "", nil
}

func applyPodPatches(pod *corev1.PodTemplateSpec, devPod *latest.DevPod) (*corev1.PodTemplateSpec, error) {
	if len(devPod.Patches) == 0 {
		return pod.DeepCopy(), nil
	}

	podBytes, err := yaml.Marshal(pod)
	if err != nil {
		return nil, err
	}

	podRaw := map[string]interface{}{}
	err = yaml.Unmarshal(podBytes, podRaw)
	if err != nil {
		return nil, err
	}

	raw, err := loader.ApplyPatchesOnObject(podRaw, devPod.Patches)
	if err != nil {
		return nil, err
	}

	// convert back
	rawJSON, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	retPod := &corev1.PodTemplateSpec{}
	err = json.Unmarshal(rawJSON, retPod)
	if err != nil {
		return nil, err
	}

	return retPod, nil
}
