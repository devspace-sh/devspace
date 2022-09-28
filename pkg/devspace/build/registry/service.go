package registry

import (
	"encoding/json"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/hash"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *LocalRegistry) ensureService(ctx devspacecontext.Context) error {
	desired := r.getService()
	raw, _ := json.Marshal(desired.Spec)
	desiredConfiguration := hash.String(string(raw))
	desired.Annotations = map[string]string{}
	desired.Annotations[LastAppliedConfigurationAnnotation] = desiredConfiguration

	// Check if there's a service already
	existing, err := ctx.KubeClient().KubeClient().CoreV1().Services(r.options.Namespace).Get(ctx.Context(), desired.Name, metav1.GetOptions{})
	if err != nil {
		// Create if not found
		if kerrors.IsNotFound(err) {
			_, err := ctx.KubeClient().KubeClient().CoreV1().Services(r.options.Namespace).Create(ctx.Context(), desired, metav1.CreateOptions{})
			if err != nil {
				return err
			}

			return nil
		}
	}

	if existing.Annotations == nil {
		existing.Annotations = map[string]string{}
	}

	// Check if configuration changes need to be applied
	lastAppliedConfiguration := existing.Annotations[LastAppliedConfigurationAnnotation]
	if desiredConfiguration != lastAppliedConfiguration {
		// Update the service
		existing.Annotations[LastAppliedConfigurationAnnotation] = desiredConfiguration
		existing.Spec = desired.Spec
		_, err := ctx.KubeClient().KubeClient().CoreV1().Services(r.options.Namespace).Update(ctx.Context(), existing, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
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
