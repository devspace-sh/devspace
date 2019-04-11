package cloud

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IngressName is the ingress name to create
const IngressName = "devspace-ingress"

// CreateIngress creates an ingress in the space if there is none
func CreateIngress(client *kubernetes.Clientset) error {
	// First check if the space has a domain
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		return errors.Wrap(err, "load generated config")
	}
	if generatedConfig.CloudSpace == nil {
		return nil
	}

	config := configutil.GetConfig()
	namespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return errors.Wrap(err, "get default namespace")
	}

	// List all ingresses and only create one if there is none already
	ingressList, err := client.ExtensionsV1beta1().Ingresses(namespace).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "list ingresses")
	}

	// Skip if there is an ingress already
	if len(ingressList.Items) > 0 {
		return nil
	}

	// Get service name and port
	serviceName := "external"
	servicePort := "80"

	for i := len(*config.Deployments) - 1; i >= 0; i-- {
		deployment := (*config.Deployments)[i]
		if deployment.Component != nil {
			if deployment.Component.Service != nil && deployment.Component.Service.Name != nil {
				serviceName = *deployment.Component.Service.Name
			} else {
				serviceName = *deployment.Name
			}

			// Get the first service port
			if deployment.Component.Service != nil && deployment.Component.Service.Ports != nil && len(*deployment.Component.Service.Ports) > 0 {
				for _, port := range *deployment.Component.Service.Ports {
					if port.Port != nil {
						servicePort = strconv.Itoa(*port.Port)
						break
					}
				}
			}

			break
		}
	}

	// Init provider
	p, err := GetProvider(&generatedConfig.CloudSpace.ProviderName, log.GetInstance())
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	// Get space
	space, err := p.GetSpace(generatedConfig.CloudSpace.SpaceID)
	if err != nil {
		return errors.Wrap(err, "get space")
	}
	if space.Domain == nil {
		return nil
	}

	// Get the cluster key
	key, err := p.GetClusterKey(space.Cluster)
	if err != nil {
		return errors.Wrap(err, "get cluster key")
	}

	// Response struct
	response := struct {
		ManagerCreateIngressPath bool `json:"manager_createKubeContextDomainIngressPath"`
	}{}

	// Do the request
	err = p.GrapqhlRequest(`
		mutation($spaceID: Int!, $ingressName: String!, $host: String!, $newPath: String!, $newServiceName: String!, $newServicePort: String!, $key: String) {
			manager_createKubeContextDomainIngressPath(
				spaceID: $spaceID,
				key: $key,
				ingressName: $ingressName,
				host: $host,
				newPath: $newPath,
				newServiceName: $newServiceName,
				newServicePort: $newServicePort,
			)
		}
	`, map[string]interface{}{
		"key":            key,
		"spaceID":        generatedConfig.CloudSpace.SpaceID,
		"ingressName":    IngressName,
		"host":           *space.Domain,
		"newPath":        "",
		"newServiceName": serviceName,
		"newServicePort": servicePort,
	}, &response)
	if err != nil {
		return errors.Wrap(err, "graphql create ingress path")
	}

	// Check result
	if response.ManagerCreateIngressPath == false {
		return errors.New("Mutation returned wrong result")
	}

	log.Infof("Successfully created ingress in space %s", generatedConfig.CloudSpace.Name)
	return nil
}
