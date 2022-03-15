package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/pkg/errors"
	"sync"
)

type Loader interface {
	// Get retrieves the kubectl client
	Get() (kubectl.Client, error)

	// UseContext recreates the kubectl client if necessary
	UseContext(context, namespace string, switchContext bool) error

	// UseNamespace recreates the kubectl client if necessary
	UseNamespace(namespace string, switchContext bool) error
}

// use_namespace [--no-switch]
// use_context --previous --ask [--no-switch]

type loader struct {
	m sync.Mutex

	defaultNamespace string
	defaultContext   string

	client kubectl.Client
}

func NewLoader(defaultNamespace, defaultContext string) Loader {
	return &loader{
		defaultNamespace: defaultNamespace,
		defaultContext:   defaultContext,
	}
}

func (l *loader) Get() (kubectl.Client, error) {
	l.m.Lock()
	defer l.m.Unlock()

	if l.client != nil {
		return l.client, nil
	}

	var err error
	l.client, err = kubectl.NewClientFromContext(l.defaultContext, l.defaultNamespace, false, kubeconfig.NewLoader())
	if err != nil {
		return nil, errors.Errorf("error creating Kubernetes client: %v. Please make sure you have a valid Kubernetes context that points to a working Kubernetes cluster. If in doubt, please check if the following command works locally: `kubectl get namespaces`", err)
	}

	return l.client, nil
}

func (l *loader) UseContext(context, namespace string, switchContext bool) error {
	l.m.Lock()
	defer l.m.Unlock()

	if l.client != nil {
		if !switchContext && l.client.Namespace() == context && l.client.Namespace() == namespace {
			return nil
		}

		l.client = nil
	}

	l.defaultNamespace = namespace
	l.defaultContext = context
	if switchContext {
		var err error
		l.client, err = kubectl.NewClientFromContext(l.defaultContext, l.defaultNamespace, switchContext, kubeconfig.NewLoader())
		if err != nil {
			return errors.Errorf("error creating Kubernetes client: %v. Please make sure you have a valid Kubernetes context that points to a working Kubernetes cluster. If in doubt, please check if the following command works locally: `kubectl get namespaces`", err)
		}
	}

	return nil
}

func (l *loader) UseNamespace(namespace string, switchContext bool) error {
	l.m.Lock()
	defer l.m.Unlock()

	if l.client != nil {
		if !switchContext && l.client.Namespace() == namespace {
			return nil
		}

		l.client = nil
	}

	l.defaultNamespace = namespace
	if switchContext {
		var err error
		l.client, err = kubectl.NewClientFromContext(l.defaultContext, l.defaultNamespace, switchContext, kubeconfig.NewLoader())
		if err != nil {
			return errors.Errorf("error creating Kubernetes client: %v. Please make sure you have a valid Kubernetes context that points to a working Kubernetes cluster. If in doubt, please check if the following command works locally: `kubectl get namespaces`", err)
		}
	}

	return nil
}
