package localregistry

import (
	"context"
	"time"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appsapplyv1 "k8s.io/client-go/applyconfigurations/apps/v1"
)

func (r *LocalRegistry) ensureStatefulset(ctx devspacecontext.Context) (*appsv1.StatefulSet, error) {
	// Switching from an unpersistent registry, delete the deployment.
	_, err := ctx.KubeClient().KubeClient().AppsV1().Deployments(r.Namespace).Get(ctx.Context(), r.Name, metav1.GetOptions{})
	if err == nil {
		err := ctx.KubeClient().KubeClient().AppsV1().Deployments(r.Namespace).Delete(ctx.Context(), r.Name, metav1.DeleteOptions{})
		if err != nil && kerrors.IsNotFound(err) {
			return nil, err
		}
	}

	var existing *appsv1.StatefulSet
	desired := r.getStatefulSet()
	kubeClient := ctx.KubeClient()
	err = wait.PollImmediateWithContext(ctx.Context(), time.Second, 30*time.Second, func(ctx context.Context) (bool, error) {
		var err error

		existing, err = kubeClient.KubeClient().AppsV1().StatefulSets(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if err == nil {
			return true, nil
		}

		if kerrors.IsNotFound(err) {
			existing, err = kubeClient.KubeClient().AppsV1().StatefulSets(r.Namespace).Create(ctx, desired, metav1.CreateOptions{})
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
	return ctx.KubeClient().KubeClient().AppsV1().StatefulSets(r.Namespace).Apply(
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
	if r.StorageClassName != "" {
		storageClassName = &r.StorageClassName
	}
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Name,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": r.Name,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: r.Name,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(r.StorageSize),
							},
						},
						StorageClassName: storageClassName,
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": r.Name,
					},
					Annotations: map[string]string{
						"container.apparmor.security.beta.kubernetes.io/buildkitd": "unconfined",
					},
				},
				Spec: corev1.PodSpec{
					EnableServiceLinks: new(bool),
					Containers:         getContainers(r.RegistryImage, r.BuildKitImage, r.Name, int32(r.Port), r.LocalBuild),
					Volumes: []corev1.Volume{
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
