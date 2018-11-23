package cloud

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"k8s.io/client-go/tools/clientcmd"
)

// DeleteDevSpace deletes the devspace from the cloud provider
func DeleteDevSpace(provider *Provider, devSpaceID string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", provider.Host+DeleteDevSpaceEndpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", provider.Token)

	if devSpaceID != "" {
		q := req.URL.Query()
		if devSpaceID != "" {
			q.Add("namespace", devSpaceID)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	} else if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("You are not allowed to delete devspace %s", devSpaceID)
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Couldn't delete devspace %s: %s. Status: %d", devSpaceID, body, resp.StatusCode)
	}

	return nil
}

// DeleteKubeContext removes the specified devspace id from the kube context if it exists
func DeleteKubeContext(devSpaceID string) error {
	config, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
	if err != nil {
		return err
	}

	hasChanged := false
	kubeContext := DevSpaceKubeContextName + "-" + devSpaceID

	if _, ok := config.Clusters[kubeContext]; ok {
		delete(config.Clusters, kubeContext)
		hasChanged = true
	}

	if _, ok := config.AuthInfos[kubeContext]; ok {
		delete(config.AuthInfos, kubeContext)
		hasChanged = true
	}

	if _, ok := config.Contexts[kubeContext]; ok {
		delete(config.Contexts, kubeContext)
		hasChanged = true
	}

	if config.CurrentContext == kubeContext {
		config.CurrentContext = ""

		if len(config.Contexts) > 0 {
			for context, contextObj := range config.Contexts {
				if contextObj != nil {
					config.CurrentContext = context
					break
				}
			}
		}

		hasChanged = true
	}

	if hasChanged {
		return kubeconfig.WriteKubeConfig(config, clientcmd.RecommendedHomeFile)
	}

	return nil
}
