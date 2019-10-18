package kubectl

import (
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
)

// GenericRequestOptions are the options for the request
type GenericRequestOptions struct {
	Resource   string
	APIVersion string

	Name          string
	LabelSelector string
	Namespace     string
}

// GenericRequest makes a new request to the given server with the specified options
func GenericRequest(client *Client, options *GenericRequestOptions) (string, error) {
	// Create new client
	var restClient restclient.Interface
	if options.APIVersion != "" {
		splitted := strings.Split(options.APIVersion, "/")
		if len(splitted) != 2 {
			return "", errors.Errorf("Error parsing %s: expected version to be group/version", options.APIVersion)
		}

		config, err := client.ClientConfig.ClientConfig()
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
		restClient = client.Client.CoreV1().RESTClient()
	}

	req := restClient.Get()
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
	out, err := req.DoRaw()
	if err != nil {
		return "", errors.Wrap(err, "request")
	}

	return string(out), nil
}
