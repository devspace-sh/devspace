package kubectl

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (client *client) CurrentContext() string {
	return client.currentContext
}

func (client *client) KubeClient() kubernetes.Interface {
	return client.Client
}

func (client *client) Host() string {
	return client.restConfig.Host
}

func (client *client) Namespace() string {
	return client.namespace
}

func (client *client) RestConfig() *rest.Config {
	return client.restConfig
}
