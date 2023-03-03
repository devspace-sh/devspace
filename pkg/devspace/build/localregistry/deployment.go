package localregistry

import (
	"context"
	"time"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
)

const BuildKitContainer = "buildkitd"

func (r *LocalRegistry) ensureDeployment(ctx devspacecontext.Context) (*appsv1.Deployment, error) {
	// Switching from a persistent registry, delete the statefulset.
	_, err := ctx.KubeClient().KubeClient().AppsV1().StatefulSets(r.Namespace).Get(ctx.Context(), r.Name, metav1.GetOptions{})
	if err == nil {
		err := ctx.KubeClient().KubeClient().AppsV1().StatefulSets(r.Namespace).Delete(ctx.Context(), r.Name, metav1.DeleteOptions{})
		if err != nil && kerrors.IsNotFound(err) {
			return nil, err
		}
	}

	// Create if it does not exist
	var existing *appsv1.Deployment
	desired := r.getDeployment()
	kubeClient := ctx.KubeClient()
	err = wait.PollImmediateWithContext(ctx.Context(), time.Second, 30*time.Second, func(ctx context.Context) (bool, error) {
		var err error

		existing, err = kubeClient.KubeClient().AppsV1().Deployments(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if err == nil {
			return true, nil
		}

		if kerrors.IsNotFound(err) {
			existing, err = kubeClient.KubeClient().AppsV1().Deployments(r.Namespace).Create(ctx, desired, metav1.CreateOptions{})
			if err == nil {
				return true, nil
			}

			if kerrors.IsAlreadyExists(err) {
				return false, nil
			}

			return false, err
		}

		return false, err
	})
	if err != nil {
		return nil, err
	}

	// Use server side apply if it does exist
	applyConfiguration, err := appsapplyv1.ExtractDeployment(existing, ApplyFieldManager)
	if err != nil {
		return nil, err
	}
	return ctx.KubeClient().KubeClient().AppsV1().Deployments(r.Namespace).Apply(
		ctx.Context(),
		applyConfiguration,
		metav1.ApplyOptions{
			FieldManager: ApplyFieldManager,
			Force:        true,
		},
	)
}

func (r *LocalRegistry) getDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": r.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": r.Name,
					},
					Annotations: getAnnotations(r.LocalBuild),
				},
				Spec: corev1.PodSpec{
					EnableServiceLinks: new(bool),
					Containers:         getContainers(r.RegistryImage, r.BuildKitImage, "registry", int32(r.Port), r.LocalBuild),
					Volumes: []corev1.Volume{
						{
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
							Name: "registry",
						},
						{
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
							Name: "buildkitd",
						},
					},
				},
			},
		},
	}
}

func getAnnotations(localbuild bool) map[string]string {
	if !localbuild {
		return map[string]string{
			"container.apparmor.security.beta.kubernetes.io/buildkitd": "unconfined",
		}
	}
	return map[string]string{}
}

// this returns a different deployment, if we're using a local docker build or not.
func getContainers(registryImage, buildKitImage, volume string, port int32, localbuild bool) []corev1.Container {
	buildContainers := getRegistryContainers(registryImage, buildKitImage, volume, port)
	if localbuild {
		// in case we're using local builds just return the deployment with only the
		// registry container inside
		return buildContainers
	}

	buildKitContainer := []corev1.Container{
		{
			Name:  BuildKitContainer,
			Image: buildKitImage,
			Args: []string{
				"--oci-worker-no-process-sandbox",
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"buildctl",
							"debug",
							"workers",
						},
					},
				},
				InitialDelaySeconds: 5,
				PeriodSeconds:       30,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"buildctl",
							"debug",
							"workers",
						},
					},
				},
				InitialDelaySeconds: 2,
				PeriodSeconds:       30,
			},
			SecurityContext: &corev1.SecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeUnconfined,
				},
				RunAsUser:  ptr.Int64(1000),
				RunAsGroup: ptr.Int64(1000),
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "buildkitd",
					MountPath: "/home/user/.local/share/buildkit",
				},
			},
		},
	}

	// in case we're using remote builds, we add the buildkit container to the
	// deployment
	return append(buildKitContainer, buildContainers...)
}

func getRegistryContainers(registryImage, buildKitImage, volume string, port int32) []corev1.Container {
	return []corev1.Container{
		{
			Name:  "registry",
			Image: registryImage,
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/v2/",
						Port: intstr.IntOrString{
							IntVal: port,
						},
					},
				},
				InitialDelaySeconds: 10,
				TimeoutSeconds:      1,
				PeriodSeconds:       20,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/v2/",
						Port: intstr.IntOrString{
							IntVal: port,
						},
					},
				},
				InitialDelaySeconds: 2,
				TimeoutSeconds:      1,
				PeriodSeconds:       5,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:                ptr.Int64(1000),
				RunAsNonRoot:             ptr.Bool(true),
				ReadOnlyRootFilesystem:   ptr.Bool(true),
				AllowPrivilegeEscalation: ptr.Bool(false),
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volume,
					MountPath: "/var/lib/registry",
				},
			},
		},
	}
}
