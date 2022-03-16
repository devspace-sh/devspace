package kubectl

import (
	"context"
	"fmt"
	"k8s.io/client-go/discovery"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
)

// GenericRequestOptions are the options for the request
type GenericRequestOptions struct {
	Kind string

	Resource   string
	APIVersion string

	Name          string
	Namespace     string
	LabelSelector string

	Method string
}

// GenericRequest makes a new request to the given server with the specified options
func (client *client) GenericRequest(ctx context.Context, options *GenericRequestOptions) (string, error) {
	// resolve Kind -> resource
	if options.Kind != "" {
		discoveryClient, err := discovery.NewDiscoveryClientForConfig(client.restConfig)
		if err != nil {
			return "", err
		}

		resources, err := discoveryClient.ServerResourcesForGroupVersion(options.APIVersion)
		if err != nil {
			return "", errors.Wrapf(err, "discover api version %s", options.APIVersion)
		}

		for _, resource := range resources.APIResources {
			if resource.Kind == options.Kind {
				options.Resource = resource.Name
				if resource.Namespaced {
					if options.Namespace == "" {
						options.Namespace = client.Namespace()
					}
				} else {
					options.Namespace = ""
				}

				break
			}
		}

		if options.Resource == "" {
			return "", fmt.Errorf("couldn't find resource for kind %s in api version %s", options.Kind, options.APIVersion)
		}
	}

	// Create new client
	var restClient restclient.Interface
	if options.APIVersion != "" && options.APIVersion != "v1" {
		splitted := strings.Split(options.APIVersion, "/")
		if len(splitted) != 2 {
			return "", errors.Errorf("Error parsing %s: expected version to be group/version", options.APIVersion)
		}

		config, err := client.ClientConfig().ClientConfig()
		if err != nil {
			return "", err
		}

		version := schema.GroupVersion{Group: splitted[0], Version: splitted[1]}
		config.GroupVersion = &version
		config.APIPath = "/apis"
		config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
		if config.UserAgent == "" {
			config.UserAgent = restclient.DefaultKubernetesUserAgent()
		}

		restClient, err = restclient.RESTClientFor(config)
		if err != nil {
			return "", err
		}
	} else {
		restClient = client.KubeClient().CoreV1().RESTClient()
	}

	var req *restclient.Request
	if options.Method == "" || options.Method == "get" {
		req = restClient.Get()
	} else if options.Method == "delete" {
		req = restClient.Delete()
	}
	if options.Namespace != "" {
		req = req.Namespace(options.Namespace)
	}
	req = req.Resource(options.Resource)
	if options.Name != "" {
		req = req.Name(options.Name)
	} else {
		reqOptions := &metav1.ListOptions{}
		if options.LabelSelector != "" {
			reqOptions.LabelSelector = options.LabelSelector
		}

		req = req.VersionedParams(reqOptions, metav1.ParameterCodec)
	}

	// Make request
	out, err := req.DoRaw(ctx)
	if err != nil {
		return "", errors.Wrap(err, "request")
	}

	return string(out), nil
}
