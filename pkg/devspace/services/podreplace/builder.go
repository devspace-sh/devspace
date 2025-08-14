package podreplace

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/kaniko/util"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/restart"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

var (
	restartHelperAnnotation = "devspace.sh/restart-helper-"

	mode = int32(0777)
)

func buildDeployment(ctx devspacecontext.Context, name string, target runtime.Object, devPod *latest.DevPod) (*appsv1.Deployment, error) {
	configHash, err := hashConfig(devPod)
	if err != nil {
		return nil, errors.Wrap(err, "hash config")
	}

	metaObject := target.(metav1.Object)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metaObject.GetNamespace(),
			Annotations: map[string]string{
				DevPodConfigHashAnnotation: configHash,
			},
			Labels: map[string]string{},
		},
	}
	for k, v := range metaObject.GetAnnotations() {
		deployment.Annotations[k] = v
	}
	for k, v := range metaObject.GetLabels() {
		deployment.Labels[k] = v
	}

	podTemplate := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: make(map[string]string),
			Labels:      make(map[string]string),
		},
	}

	switch t := target.(type) {
	case *appsv1.ReplicaSet:
		deployment.Annotations[TargetNameAnnotation] = t.Name
		deployment.Annotations[TargetKindAnnotation] = "ReplicaSet"
		deployment.Spec.Selector = t.Spec.Selector
		podTemplate.Labels = t.Spec.Template.Labels
		podTemplate.Annotations = t.Spec.Template.Annotations
		podTemplate.Spec = *t.Spec.Template.Spec.DeepCopy()
	case *appsv1.Deployment:
		deployment.Annotations[TargetNameAnnotation] = t.Name
		deployment.Annotations[TargetKindAnnotation] = "Deployment"
		deployment.Spec.Selector = t.Spec.Selector
		podTemplate.Labels = t.Spec.Template.Labels
		podTemplate.Annotations = t.Spec.Template.Annotations
		podTemplate.Spec = *t.Spec.Template.Spec.DeepCopy()
	case *appsv1.StatefulSet:
		deployment.Annotations[TargetNameAnnotation] = t.Name
		deployment.Annotations[TargetKindAnnotation] = "StatefulSet"
		deployment.Spec.Selector = t.Spec.Selector
		podTemplate.Labels = t.Spec.Template.Labels
		podTemplate.Annotations = t.Spec.Template.Annotations
		podTemplate.Spec = *t.Spec.Template.Spec.DeepCopy()
		podTemplate.Spec.Hostname = strings.ReplaceAll(t.Name+"-0", ".", "-")
		for _, pvc := range t.Spec.VolumeClaimTemplates {
			volumeName := pvc.ObjectMeta.Name
			if volumeName == "" {
				volumeName = "data"
			}
			claimName := volumeName + "-" + t.Name + "-0"
			podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: claimName,
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
	}

	// apply the patches
	podTemplate, err = applyPodPatches(podTemplate, devPod)
	if err != nil {
		return nil, errors.Wrap(err, "apply pod patches")
	}

	// check if terminal and modify pod
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		err = modifyDevContainer(ctx, devPod, devContainer, podTemplate)
		return err == nil
	})
	if err != nil {
		return nil, err
	}

	// replace paths
	persist := false
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		persist = len(devContainer.PersistPaths) > 0
		return !persist
	})
	if persist {
		err := persistPaths(name, devPod, podTemplate)
		if err != nil {
			return nil, err
		}
	}

	// reset the metadata
	if podTemplate.Labels == nil {
		podTemplate.Labels = map[string]string{}
	}
	if podTemplate.Annotations == nil {
		podTemplate.Annotations = map[string]string{}
	}
	deployment.Labels[selector.ReplacedLabel] = "true"
	podTemplate.Labels[selector.ReplacedLabel] = "true"
	imageSelector, err := hashImageSelector(ctx, devPod)
	if err != nil {
		return nil, err
	} else if imageSelector != "" {
		podTemplate.Annotations[selector.ImageSelectorAnnotation] = imageSelector
	}
	if len(containers) > 0 {
		deployment.Annotations[selector.MatchedContainerAnnotation] = strings.Join(containers, ";")
		podTemplate.Annotations[selector.MatchedContainerAnnotation] = strings.Join(containers, ";")
	}

	deployment.Spec.Template = *podTemplate
	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = &metav1.LabelSelector{}
	}
	if deployment.Spec.Selector.MatchLabels == nil {
		deployment.Spec.Selector.MatchLabels = podTemplate.Labels
		deployment.Spec.Selector.MatchExpressions = nil
	}
	deployment.Spec.Selector.MatchLabels[selector.ReplacedLabel] = "true"
	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}

	// make sure labels etc are there
	if ctx.Log().GetLevel() == logrus.DebugLevel {
		out, _ := yaml.Marshal(podTemplate)
		ctx.Log().Debugf("Replaced pod spec: \n%v\n", string(out))
	}

	return deployment, nil
}

func modifyDevContainer(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) error {
	err := replaceImage(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return err
	}

	err = replaceTerminal(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return errors.Wrap(err, "replace terminal")
	}

	err = replaceAttach(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return errors.Wrap(err, "replace attach")
	}

	err = replaceEnv(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return errors.Wrap(err, "replace env")
	}

	err = replaceCommand(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return errors.Wrap(err, "replace entrypoint")
	}

	err = replaceWorkingDir(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return errors.Wrap(err, "replace working dir")
	}

	err = replaceSecurityContext(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return errors.Wrap(err, "replace securitycontext")
	}

	err = replaceResources(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return errors.Wrap(err, "replace resources")
	}

	return nil
}

func replaceResources(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) error {
	if devContainer.Resources == nil {
		return nil
	}

	index, container, err := getPodTemplateContainer(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return err
	}

	limits, err := util.ConvertMap(devContainer.Resources.Limits)
	if err != nil {
		return errors.Wrap(err, "parse limits")
	}

	requests, err := util.ConvertMap(devContainer.Resources.Requests)
	if err != nil {
		return errors.Wrap(err, "parse requests")
	}

	container.Resources.Limits = limits
	container.Resources.Requests = requests
	podTemplate.Spec.Containers[index] = *container
	return nil
}

func replaceWorkingDir(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) error {
	if devContainer.WorkingDir == "" {
		return nil
	}

	index, container, err := getPodTemplateContainer(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return err
	}

	container.ReadinessProbe = nil
	container.LivenessProbe = nil
	container.StartupProbe = nil
	container.WorkingDir = devContainer.WorkingDir
	podTemplate.Spec.Containers[index] = *container
	return nil
}

func replaceSecurityContext(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) error {
	if devContainer.Sync == nil {
		return nil
	}

	index, container, err := getPodTemplateContainer(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return err
	}

	if container.SecurityContext != nil {
		container.SecurityContext.ReadOnlyRootFilesystem = nil
		podTemplate.Spec.Containers[index] = *container
	}
	return nil
}

func replaceCommand(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) error {
	// replace with DevSpace helper
	injectRestartHelper := devContainer.RestartHelper != nil && devContainer.RestartHelper.Inject != nil && *devContainer.RestartHelper.Inject
	if devContainer.RestartHelper == nil || devContainer.RestartHelper.Inject == nil || *devContainer.RestartHelper.Inject {
		for _, s := range devContainer.Sync {
			if s.StartContainer || (s.OnUpload != nil && s.OnUpload.RestartContainer) {
				injectRestartHelper = true
			}
		}
	}
	if len(devContainer.Command) == 0 && injectRestartHelper {
		return fmt.Errorf("dev.%s.sync[*].onUpload.restartContainer or dev.%s.restartHelper.inject is true, please specify the entrypoint that should get restarted in dev.%s.command", devPod.Name, devPod.Name, devPod.Name)
	}
	if !injectRestartHelper && len(devContainer.Command) == 0 && devContainer.Args == nil {
		return nil
	}

	index, container, err := getPodTemplateContainer(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return err
	}

	// make sure probes are not set for this container
	container.ReadinessProbe = nil
	container.LivenessProbe = nil
	container.StartupProbe = nil

	// should we inject devspace restart helper?
	if injectRestartHelper {
		containerHash := strings.ToLower(hash.String(container.Name))[0:10]
		annotationName := restartHelperAnnotation + containerHash
		if podTemplate.Annotations == nil {
			podTemplate.Annotations = map[string]string{}
		}

		restartHelperPath := ""
		if devContainer.RestartHelper != nil {
			restartHelperPath = devContainer.RestartHelper.Path
		}

		restartHelperString, err := restart.LoadRestartHelper(restartHelperPath)
		if err != nil {
			return errors.Wrap(err, "load restart helper")
		}
		podTemplate.Annotations[annotationName] = restartHelperString

		volumeName := "devspace-restart-" + containerHash
		podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				DownwardAPI: &corev1.DownwardAPIVolumeSource{
					DefaultMode: &mode,
					Items: []corev1.DownwardAPIVolumeFile{
						{
							Path: restart.ScriptName,
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "metadata.annotations['" + annotationName + "']",
							},
							Mode: &mode,
						},
					},
				},
			},
		})
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  true,
			SubPath:   restart.ScriptName,
			MountPath: restart.ScriptPath,
		})

		container.Command = []string{restart.ScriptPath}
		container.Command = append(container.Command, devContainer.Command...)
		if devContainer.Args != nil {
			container.Args = devContainer.Args
		}
		podTemplate.Spec.Containers[index] = *container
		return nil
	}

	if len(devContainer.Command) > 0 {
		container.Command = devContainer.Command
	}
	if devContainer.Args != nil {
		container.Args = devContainer.Args
	}
	podTemplate.Spec.Containers[index] = *container
	return nil
}

func replaceEnv(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) error {
	if len(devContainer.Env) == 0 {
		return nil
	}

	index, container, err := getPodTemplateContainer(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return err
	}

	for _, v := range devContainer.Env {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  v.Name,
			Value: v.Value,
		})
	}

	podTemplate.Spec.Containers[index] = *container
	return nil
}

func replaceAttach(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) error {
	if devContainer.Attach == nil || devContainer.Attach.DisableReplace || (devContainer.Attach.Enabled != nil && !*devContainer.Attach.Enabled) {
		return nil
	}

	index, container, err := getPodTemplateContainer(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return err
	}

	container.ReadinessProbe = nil
	container.StartupProbe = nil
	container.LivenessProbe = nil
	container.Stdin = true
	container.TTY = !devContainer.Attach.DisableTTY
	podTemplate.Spec.Containers[index] = *container
	return nil
}

func replaceTerminal(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) error {
	if devContainer.Terminal == nil || devContainer.Terminal.DisableReplace || (devContainer.Terminal.Enabled != nil && !*devContainer.Terminal.Enabled) {
		return nil
	}

	index, container, err := getPodTemplateContainer(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return err
	}

	container.ReadinessProbe = nil
	container.StartupProbe = nil
	container.LivenessProbe = nil
	container.Command = []string{"sleep", "1000000000"}
	container.Args = []string{}
	podTemplate.Spec.Containers[index] = *container
	return nil
}

func getPodTemplateContainer(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) (int, *corev1.Container, error) {
	containerName := devContainer.Container
	if containerName == "" && len(podTemplate.Spec.Containers) > 1 {
		containers, err := matchesImageSelector(ctx, podTemplate, devPod)
		if err != nil {
			return 0, nil, err
		} else if len(containers) != 1 {
			names := []string{}
			for _, c := range podTemplate.Spec.Containers {
				names = append(names, c.Name)
			}

			return 0, nil, fmt.Errorf("couldn't modify pod as multiple containers were found '%s', but no dev.*.container was specified", strings.Join(names, "' '"))
		}

		containerName = containers[0]
	}

	for i, con := range podTemplate.Spec.Containers {
		if containerName == "" || con.Name == containerName {
			return i, &con, nil
		}
	}

	return 0, nil, fmt.Errorf("couldn't find container '%s' in pod", containerName)
}

func hashConfig(replacePod *latest.DevPod) (string, error) {
	out, err := yaml.Marshal(replacePod)
	if err != nil {
		return "", err
	}

	return hash.String(string(out)), nil
}

func replaceImage(ctx devspacecontext.Context, devPod *latest.DevPod, devContainer *latest.DevContainer, podTemplate *corev1.PodTemplateSpec) error {
	if devContainer.DevImage == "" {
		return nil
	}

	index, container, err := getPodTemplateContainer(ctx, devPod, devContainer, podTemplate)
	if err != nil {
		return err
	}

	imageStr, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir(), true).FillRuntimeVariablesAsString(ctx.Context(), devContainer.DevImage, ctx.Config(), ctx.Dependencies())
	if err != nil {
		return err
	}

	container.ReadinessProbe = nil
	container.LivenessProbe = nil
	container.StartupProbe = nil
	container.Image = imageStr
	podTemplate.Spec.Containers[index] = *container
	return nil
}

func hashImageSelector(ctx devspacecontext.Context, replacePod *latest.DevPod) (string, error) {
	if replacePod.ImageSelector != "" {
		imageSelector, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir(), true).FillRuntimeVariablesAsImageSelector(ctx.Context(), replacePod.ImageSelector, ctx.Config(), ctx.Dependencies())
		if err != nil {
			return "", err
		} else if imageSelector == nil {
			return "", fmt.Errorf("couldn't resolve image selector: %v", replacePod.ImageSelector)
		}

		return imageSelector.Image, nil
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
	err = yaml.Unmarshal(podBytes, &podRaw)
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

	retPod := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: make(map[string]string),
			Labels:      make(map[string]string),
		},
	}

	err = json.Unmarshal(rawJSON, retPod)
	if err != nil {
		return nil, err
	}
	return retPod, nil
}
