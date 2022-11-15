package registry

import (
	"context"
	"time"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
)

func (r *LocalRegistry) ensureStatefulset(ctx devspacecontext.Context) (*appsv1.StatefulSet, error) {
	// Switching from an unpersistent registry, delete the deployment.
	_, err := ctx.KubeClient().KubeClient().AppsV1().Deployments(r.options.Namespace).Get(ctx.Context(), r.options.Name, metav1.GetOptions{})
	if err == nil {
		err := ctx.KubeClient().KubeClient().AppsV1().Deployments(r.options.Namespace).Delete(ctx.Context(), r.options.Name, metav1.DeleteOptions{})
		if err != nil && kerrors.IsNotFound(err) {
			return nil, err
		}
	}

	var existing *appsv1.StatefulSet
	desired := r.getStatefulSet()
	kubeClient := ctx.KubeClient()
	err = wait.PollImmediateWithContext(ctx.Context(), time.Second, 30*time.Second, func(ctx context.Context) (bool, error) {
		var err error

		existing, err = kubeClient.KubeClient().AppsV1().StatefulSets(r.options.Namespace).Get(ctx, r.options.Name, metav1.GetOptions{})
		if err == nil {
			return true, nil
		}

		if kerrors.IsNotFound(err) {
			existing, err = kubeClient.KubeClient().AppsV1().StatefulSets(r.options.Namespace).Create(ctx, desired, metav1.CreateOptions{})
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
	applyConfiguration, err := appsapplyv1.ExtractStatefulSet(existing, ApplyFieldManager)
	if err != nil {
		return nil, err
	}
	return ctx.KubeClient().KubeClient().AppsV1().StatefulSets(r.options.Namespace).Apply(
		ctx.Context(),
		applyConfiguration,
		metav1.ApplyOptions{
			FieldManager: ApplyFieldManager,
			Force:        true,
		},
	)
}

func (r *LocalRegistry) getStatefulSet() *appsv1.StatefulSet {
	var storageClassName *string
	if r.options.StorageClassName != "" {
		storageClassName = &r.options.StorageClassName
	}
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.options.Name,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": r.options.Name,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: r.options.Name,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(r.options.StorageSize),
							},
						},
						StorageClassName: storageClassName,
					},
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
									TCPSocket: &corev1.TCPSocketAction{
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
									TCPSocket: &corev1.TCPSocketAction{
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
									Name:      r.options.Name,
									MountPath: "/var/lib/registry",
								},
							},
						},
					},
				},
			},
		},
	}
}
