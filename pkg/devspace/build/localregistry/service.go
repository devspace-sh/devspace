package localregistry

import (
	"context"
	"time"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
)

func (r *LocalRegistry) ensureService(ctx devspacecontext.Context) (*corev1.Service, error) {
	// Create if it does not exist
	var existing *corev1.Service
	desired := r.getService()
	kubeClient := ctx.KubeClient()
	err := wait.PollImmediateWithContext(ctx.Context(), time.Second, 30*time.Second, func(ctx context.Context) (bool, error) {
		var err error

		existing, err = kubeClient.KubeClient().CoreV1().Services(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if err == nil {
			return true, nil
		}

		if kerrors.IsNotFound(err) {
			existing, err = kubeClient.KubeClient().CoreV1().Services(r.Namespace).Create(ctx, desired, metav1.CreateOptions{})
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
	applyConfiguration, err := applyv1.ExtractService(existing, ApplyFieldManager)
	if err != nil {
		return nil, err
	}

	return ctx.KubeClient().KubeClient().CoreV1().Services(r.Namespace).Apply(
		ctx.Context(),
		applyConfiguration,
		metav1.ApplyOptions{
			FieldManager: ApplyFieldManager,
			Force:        true,
		},
	)
}

func (r *LocalRegistry) getService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "registry",
					Protocol: corev1.ProtocolTCP,
					Port:     int32(r.Port),
					TargetPort: intstr.IntOrString{
						IntVal: int32(r.Port),
					},
				},
			},
			Selector: map[string]string{
				"app": r.Name,
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}
}
