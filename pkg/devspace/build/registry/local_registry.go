package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
)

const (
	LastAppliedConfigurationAnnotation = "devspace.sh/last-applied-configuration"
	ApplyFieldManager                  = "devspace"
)

type LocalRegistry struct {
	options     Options
	host        string
	servicePort *corev1.ServicePort
}

func NewLocalRegistry(options Options) *LocalRegistry {
	return &LocalRegistry{
		options: options,
	}
}

func (r *LocalRegistry) IsStarted() bool {
	return r.servicePort != nil
}

func (r *LocalRegistry) Start(ctx devspacecontext.Context) error {
	ctx.Log().Info("Starting Local Image Registry")

	if err := r.ensureNamespace(ctx); err != nil {
		return errors.Wrap(err, "ensure namespace")
	}

	if r.options.StorageEnabled {
		if _, err := r.ensureStatefulset(ctx); err != nil {
			return errors.Wrap(err, "ensure statefulset")
		}
	} else {
		if _, err := r.ensureDeployment(ctx); err != nil {
			return errors.Wrap(err, "ensure deployment")
		}
	}

	if _, err := r.ensureService(ctx); err != nil {
		return errors.Wrap(err, "ensure service")
	}

	// Wait for service to have a node port
	var err error
	r.servicePort, err = r.waitForNodePort(ctx)
	if err != nil {
		return errors.Wrap(err, "wait for node port")
	}

	// Save registry host for rewriting images
	r.host = fmt.Sprintf("localhost:%d", r.servicePort.NodePort)

	// Select the registry pod
	imageRegistryPod, err := r.selectRegistryPod(ctx)
	if err != nil {
		return errors.Wrap(err, "select registry pod")
	}

	// Check if local registry is already available
	isRegistryAvailable, err := r.ping(ctx.Context())
	if err != nil {
		return errors.Wrap(err, "ping local registry")
	}

	if !isRegistryAvailable {
		// Start port forwarding
		ctx.Log().Debug("Starting local registry port forwarding")
		if err := r.startPortForwarding(ctx, imageRegistryPod); err != nil {
			return errors.Wrap(err, "start port forwarding")
		}
	} else {
		ctx.Log().Debug("Skip local registry port forwarding")
	}

	// Wait for registry to be responsive
	if err := r.waitForRegistry(ctx.Context()); err != nil {
		return errors.Wrap(err, "wait for registry")
	}

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

func (r *LocalRegistry) ensureNamespace(ctx devspacecontext.Context) error {
	applyConfiguration, err := applyv1.ExtractNamespace(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.options.Namespace,
		},
	}, ApplyFieldManager)
	if err != nil {
		return err
	}

	_, err = ctx.KubeClient().KubeClient().CoreV1().Namespaces().Apply(
		ctx.Context(),
		applyConfiguration,
		metav1.ApplyOptions{
			FieldManager: ApplyFieldManager,
			Force:        true,
		},
	)
	return err
}

func (r *LocalRegistry) ping(ctx context.Context) (done bool, err error) {
	registry, err := name.NewRegistry(r.host)
	if err != nil {
		return false, err
	}

	_, err = remote.Catalog(ctx, registry)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (r *LocalRegistry) selectRegistryPod(ctx devspacecontext.Context) (*corev1.Pod, error) {
	options := targetselector.NewEmptyOptions().
		WithLabelSelector(fmt.Sprintf("app=%s", r.options.Name)).
		WithNamespace(r.options.Namespace).
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Millisecond * 500)).
		WithSkipInitContainers(true)
	selector := targetselector.NewTargetSelector(options)
	return selector.SelectSinglePod(ctx.Context(), ctx.KubeClient(), &log.DiscardLogger{})
}

func (r *LocalRegistry) startPortForwarding(ctx devspacecontext.Context, imageRegistryPod *corev1.Pod) error {
	localPort := r.servicePort.NodePort
	remotePort := r.servicePort.TargetPort.IntVal
	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}
	addresses := []string{"localhost"}
	portsFormatted := ansi.Color(fmt.Sprintf("%d -> %d", int(localPort), int(remotePort)), "white+b")
	readyChan := make(chan struct{})
	errorChan := make(chan error, 1)
	pf, err := kubectl.NewPortForwarder(
		ctx.KubeClient(),
		imageRegistryPod,
		ports,
		addresses,
		make(chan struct{}),
		readyChan,
		errorChan,
	)
	if err != nil {
		return errors.Errorf("Error starting port forwarding: %v", err)
	}

	go func() {
		err := pf.ForwardPorts(ctx.Context())
		if err != nil {
			errorChan <- err
		}
	}()

	select {
	case <-ctx.Context().Done():
		ctx.Log().Donef("Port forwarding to local registry stopped")
		return nil
	case <-readyChan:
		ctx.Log().Donef("Port forwarding to local registry started on: %s", portsFormatted)
	case err := <-errorChan:
		if ctx.IsDone() {
			return nil
		}

		return errors.Wrap(err, "forward ports")
	case <-time.After(20 * time.Second):
		return errors.Errorf("Timeout waiting for port forwarding to start")
	}

	return nil
}

func (r *LocalRegistry) waitForNodePort(ctx devspacecontext.Context) (*corev1.ServicePort, error) {
	var servicePort *corev1.ServicePort

	kubeClient := ctx.KubeClient().KubeClient()
	err := wait.PollImmediateWithContext(ctx.Context(), time.Second, 30*time.Second, func(ctx context.Context) (done bool, err error) {
		service, err := kubeClient.CoreV1().Services(r.options.Namespace).Get(ctx, r.options.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		servicePort = GetServicePort(service)

		return servicePort.NodePort != 0, nil
	})

	return servicePort, err
}

func (r *LocalRegistry) waitForRegistry(ctx context.Context) error {
	return wait.PollImmediateWithContext(ctx, time.Second, 30*time.Second, func(ctx context.Context) (done bool, err error) {
		return r.ping(ctx)
	})
}
