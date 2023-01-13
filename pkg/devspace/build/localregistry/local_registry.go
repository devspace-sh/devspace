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

func GetOrCreateLocalRegistry(ctx devspacecontext.Context, options Options) (*LocalRegistry, error) {
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

	// Select the registry pod
	ctx.Log().Debug("Wait for running local registry pod...")
	_, err = r.SelectRegistryPod(ctx)
	if err != nil {
		return errors.Wrap(err, "select registry pod")
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
	err := wait.PollImmediateWithContext(ctx.Context(), time.Second, 30*time.Second, func(ctx context.Context) (done bool, err error) {
		service, err := kubeClient.CoreV1().Services(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		servicePort = GetServicePort(service)
		return servicePort.NodePort != 0, nil
	})

	return servicePort, err
}
