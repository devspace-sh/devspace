package registry

import (
	"encoding/json"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *LocalRegistry) ensureDeployment(ctx devspacecontext.Context) error {
	// Switching from a persistent registry, delete the statefulset.
	_, err := ctx.KubeClient().KubeClient().AppsV1().StatefulSets(r.options.Namespace).Get(ctx.Context(), r.options.Name, metav1.GetOptions{})
	if err == nil {
		err := ctx.KubeClient().KubeClient().AppsV1().StatefulSets(r.options.Namespace).Delete(ctx.Context(), r.options.Name, metav1.DeleteOptions{})
		if err != nil && kerrors.IsNotFound(err) {
			return err
		}
	}

	desired := r.getDeployment()
	raw, _ := json.Marshal(desired.Spec)
	desiredConfiguration := hash.String(string(raw))
	desired.Annotations = map[string]string{}
	desired.Annotations[LastAppliedConfigurationAnnotation] = desiredConfiguration

	// Check if there's a deployment already
	existing, err := ctx.KubeClient().KubeClient().AppsV1().Deployments(r.options.Namespace).Get(ctx.Context(), r.options.Name, metav1.GetOptions{})
	if err != nil {
		// Create if not found
		if kerrors.IsNotFound(err) {
			_, err = ctx.KubeClient().KubeClient().AppsV1().Deployments(r.options.Namespace).Create(ctx.Context(), desired, metav1.CreateOptions{})
			if err != nil {
				return err
			}

			return nil
		}

		return err
	}

	if existing.Annotations == nil {
		existing.Annotations = map[string]string{}
	}

	// Check if configuration changes need to be applied
	lastAppliedConfiguration := existing.Annotations[LastAppliedConfigurationAnnotation]
	if desiredConfiguration != lastAppliedConfiguration {
		// Update the deployment
		existing.Annotations[LastAppliedConfigurationAnnotation] = desiredConfiguration
		existing.Spec = desired.Spec
		_, err := ctx.KubeClient().KubeClient().AppsV1().Deployments(r.options.Namespace).Update(ctx.Context(), existing, metav1.UpdateOptions{})
		if err != nil {
			// Re-create if update fails
			if kerrors.IsInvalid(err) {
				err := ctx.KubeClient().KubeClient().AppsV1().Deployments(r.options.Namespace).Delete(ctx.Context(), existing.Name, metav1.DeleteOptions{})
				if err != nil {
					return err
				}

				_, err = ctx.KubeClient().KubeClient().AppsV1().Deployments(r.options.Namespace).Create(ctx.Context(), desired, metav1.CreateOptions{})
				if err != nil {
					return err
				}

				return nil
			}

			return err
		}
	}

	return nil
}

func (r *LocalRegistry) getDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.options.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": r.options.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": r.options.Name,
					},
				},
				Spec: corev1.PodSpec{
					EnableServiceLinks: new(bool),
					Containers: []corev1.Container{
						{
							Name:  "registry",
							Image: r.options.Image,
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/v2/",
										Port: intstr.IntOrString{
											IntVal: int32(r.options.Port),
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
											IntVal: int32(r.options.Port),
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
								AllowPrivilegeEscalation: new(bool),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "registry",
									MountPath: "/var/lib/registry",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
							Name: "registry",
						},
					},
				},
			},
		},
	}
}
