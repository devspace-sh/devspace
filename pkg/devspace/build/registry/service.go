package registry

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
)

func (r *LocalRegistry) ensureService(ctx devspacecontext.Context) (*corev1.Service, error) {
	// Create if it does not exist
	desired := r.getService()
	existing, err := ctx.KubeClient().KubeClient().CoreV1().Services(r.options.Namespace).Get(ctx.Context(), r.options.Name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctx.KubeClient().KubeClient().CoreV1().Services(r.options.Namespace).Create(ctx.Context(), desired, metav1.CreateOptions{})
		}

		return nil, err
	}

	// Use server side apply if it does exist
	applyConfiguration, err := applyv1.ExtractService(existing, ApplyFieldManager)
	if err != nil {
		return nil, err
	}

	return ctx.KubeClient().KubeClient().CoreV1().Services(r.options.Namespace).Apply(
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
			Name: r.options.Name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "registry",
					Protocol: corev1.ProtocolTCP,
					Port:     int32(r.options.Port),
					TargetPort: intstr.IntOrString{
						IntVal: int32(r.options.Port),
					},
				},
			},
			Selector: map[string]string{
				"app": r.options.Name,
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}
}
