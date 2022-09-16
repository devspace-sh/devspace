package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/portforwarding"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

type LocalRegistry struct {
	options Options
	host    string
}

func NewLocalRegistry(options Options) *LocalRegistry {
	return &LocalRegistry{
		options: options,
	}
}

func (r *LocalRegistry) Start(ctx devspacecontext.Context) error {
	ctx.Log().Info("Starting Local Image Registry")

	// Persistence enabled
	client := ctx.KubeClient()
	if r.options.StorageEnabled {
		// Create StatefulSet
		_, err := client.KubeClient().AppsV1().StatefulSets(r.options.Namespace).Create(ctx.Context(), r.getStatefulSet(), metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "create statefulset")
		}
	} else {
		// Create Deployment
		_, err := client.KubeClient().AppsV1().Deployments(r.options.Namespace).Create(ctx.Context(), r.getDeployment(), metav1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "create statefulset")
		}
	}

	// Create service
	_, err := client.KubeClient().CoreV1().Services(r.options.Namespace).Create(ctx.Context(), r.getService(), metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "create service")
	}

	// Wait for registry
	ctx.Log().Infof("Waiting for Local Image Registry to become ready...")
	options := targetselector.NewEmptyOptions().
		WithLabelSelector(fmt.Sprintf("app=%s", r.options.Name)).
		WithNamespace(r.options.Namespace).
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Millisecond * 500)).
		WithSkipInitContainers(true)
	selector := targetselector.NewTargetSelector(options)
	imageRegistryPod, err := selector.SelectSinglePod(ctx.Context(), ctx.KubeClient(), &log.DiscardLogger{})
	if err != nil {
		return errors.Wrap(err, "wait for registry")
	}

	// Wait for service to have endpoints
	err = wait.PollImmediateWithContext(ctx.Context(), time.Second, 30*time.Second, func(ctx context.Context) (done bool, err error) {
		endpoints, err := client.KubeClient().CoreV1().Endpoints(r.options.Namespace).Get(ctx, r.options.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		ready := true
		for _, subset := range endpoints.Subsets {
			if len(subset.NotReadyAddresses) > 0 {
				ready = false
				break
			}
		}

		return ready, nil
	})
	if err != nil {
		return err
	}

	// Get node port of service
	service, err := client.KubeClient().CoreV1().Services(r.options.Namespace).Get(ctx.Context(), r.options.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get service")
	}
	nodePort := GetNodePort(service)
	r.host = fmt.Sprintf("localhost:%d", nodePort)

	// Start port forwarding
	portForwardCtx, parent := ctx.WithNewTomb()
	portForwardDone := parent.NotifyGo(func() error {
		return portforwarding.StartForwarding(
			portForwardCtx,
			imageRegistryPod.Name,
			[]*latest.PortMapping{{
				Port: fmt.Sprintf("%d", nodePort),
			}},
			selector,
			parent,
		)
	})
	if !parent.Alive() {
		return errors.Wrap(parent.Err(), "portforward local registry")
	}

	<-portForwardDone

	return nil
}

func (r *LocalRegistry) RewriteImage(image string) (string, error) {
	registry, err := name.NewRegistry(r.host)
	if err != nil {
		return "", err
	}

	tag, err := name.NewTag(image)
	if err != nil {
		return "", err
	}

	tag.Registry = registry

	return tag.Repository.Name(), nil
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
									Name:      "registry",
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
