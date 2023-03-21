package localregistry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/mgutz/ansi"
)

var (
	localRegistries     = map[string]*LocalRegistry{}
	localRegistriesLock sync.Mutex
)

const (
	ApplyFieldManager = "devspace"
)

type LocalRegistry struct {
	Options
	host        string
	servicePort *corev1.ServicePort
}

func GetOrCreateLocalRegistry(
	ctx devspacecontext.Context,
	options Options,
) (*LocalRegistry, error) {
	localRegistriesLock.Lock()
	defer localRegistriesLock.Unlock()

	id := getID(options)
	localRegistry := localRegistries[id]
	if localRegistry == nil {
		localRegistry = newLocalRegistry(options)
		ctx := ctx.WithLogger(ctx.Log().
			WithPrefix("local-registry: ")).
			WithContext(context.Background())

		err := localRegistry.Start(ctx)
		if err != nil {
			return nil, err
		}
		localRegistries[id] = localRegistry
	}

	return localRegistry, nil
}

func newLocalRegistry(options Options) *LocalRegistry {
	return &LocalRegistry{
		Options: options,
	}
}

func (r *LocalRegistry) Start(ctx devspacecontext.Context) error {
	ctx.Log().Info("Starting Local Image Registry")

	if err := r.ensureNamespace(ctx); err != nil {
		return errors.Wrap(err, "ensure namespace")
	}

	if r.StorageEnabled {
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
	ctx.Log().Debug("Wait for local registry node port to be assigned...")
	var err error
	r.servicePort, err = r.waitForNodePort(ctx)
	if err != nil {
		return errors.Wrap(err, "wait for node port")
	}

	// Save registry host for rewriting images
	r.host = fmt.Sprintf("localhost:%d", r.servicePort.NodePort)

	// Check if local registry is already available
	ctx.Log().Debug("Check for running local registry")
	isRegistryAvailable, err := r.ping(ctx.Context())
	if err != nil {
		return errors.Wrap(err, "ping local registry")
	}

	// Select the registry pod
	ctx.Log().Debug("Wait for running local registry pod...")
	imageRegistryPod, err := r.SelectRegistryPod(ctx)
	if err != nil {
		return errors.Wrap(err, "select registry pod")
	}

	if r.LocalBuild {
		// In case of local builds, we'll need to start registry port forwarding
		// in order to push images from local builds to cluster's registry
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
		ctx.Log().Debug("Waiting for local registry to become ready...")
		if err := r.waitForRegistry(ctx.Context()); err != nil {
			return errors.Wrap(err, "wait for registry")
		}
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

func (r *LocalRegistry) RewriteImageForBuilder(image string) (string, error) {
	registry, err := name.NewRegistry("localhost:5000")
	if err != nil {
		return "", err
	}

	tag, err := name.NewTag(image)
	if err != nil {
		return "", err
	}

	tag.Registry = registry
	return tag.Name(), nil
}

func (r *LocalRegistry) ensureNamespace(ctx devspacecontext.Context) error {
	// If localregistry namespace is the same as devspace, we don't have
	// anything to do.
	if r.Namespace == ctx.KubeClient().Namespace() {
		return nil
	}

	// List all namespaces, this will already return an error in case of
	// user's permissions problems.
	namespaces, err := ctx.KubeClient().
		KubeClient().
		CoreV1().
		Namespaces().
		List(ctx.Context(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	// Check if the namespace already is there, if not we'll try to create it.
	for _, namespace := range namespaces.Items {
		if r.Namespace == namespace.Name {
			ctx.Log().Debugf("Namespace %s already exists, skipping creation", r.Namespace)
			return nil
		}
	}

	ctx.Log().Debugf("Namespace %s doesn't exist, attempting creation", r.Namespace)
	applyConfiguration, err := applyv1.ExtractNamespace(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Namespace,
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

func (r *LocalRegistry) SelectRegistryPod(ctx devspacecontext.Context) (*corev1.Pod, error) {
	options := targetselector.NewEmptyOptions().
		WithLabelSelector(fmt.Sprintf("app=%s", r.Name)).
		WithNamespace(r.Namespace).
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Millisecond * 500)).
		WithSkipInitContainers(true)
	selector := targetselector.NewTargetSelector(options)
	return selector.SelectSinglePod(ctx.Context(), ctx.KubeClient(), &log.DiscardLogger{})
}

func (r *LocalRegistry) waitForNodePort(ctx devspacecontext.Context) (*corev1.ServicePort, error) {
	var servicePort *corev1.ServicePort

	kubeClient := ctx.KubeClient().KubeClient()
	err := wait.PollImmediateWithContext(
		ctx.Context(),
		time.Second,
		30*time.Second,
		func(ctx context.Context) (done bool, err error) {
			service, err := kubeClient.CoreV1().
				Services(r.Namespace).
				Get(ctx, r.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			servicePort = GetServicePort(service)
			return servicePort.NodePort != 0, nil
		},
	)

	return servicePort, err
}

// GetRegistryURL returns the host:port of the current registry
func (r *LocalRegistry) GetRegistryURL() string {
	return r.host
}

// startPortForwarding will forward container's port into localhost in order to access registry's container in
// the cluster, locally, to push the built image afterwards
func (r *LocalRegistry) startPortForwarding(
	ctx devspacecontext.Context,
	imageRegistryPod *corev1.Pod,
) error {
	localPort := r.servicePort.NodePort
	remotePort := r.servicePort.TargetPort.IntVal
	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}
	addresses := []string{"localhost"}
	portsFormatted := ansi.Color(
		fmt.Sprintf("%d -> %d", int(localPort), int(remotePort)),
		"white+b",
	)
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

func (r *LocalRegistry) waitForRegistry(ctx context.Context) error {
	return wait.PollImmediateWithContext(
		ctx,
		time.Second,
		30*time.Second,
		func(ctx context.Context) (done bool, err error) {
			return r.ping(ctx)
		},
	)
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
