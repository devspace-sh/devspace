package deploy

import (
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/deploy/helm"
	"github.com/covexo/devspace/pkg/devspace/deploy/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"k8s.io/client-go/kubernetes"
)

// All deploys all deployments in the config
func All(client *kubernetes.Clientset, generatedConfig *generated.Config, forceDeploy, useDevOverwrite bool, log log.Logger) error {
	config := configutil.GetConfig()

	if config.DevSpace.Deployments != nil {
		for _, deployConfig := range *config.DevSpace.Deployments {
			var deployClient Interface
			var err error

			if deployConfig.Kubectl != nil {
				log.Info("Deploying " + *deployConfig.Name + " with kubectl")

				deployClient, err = kubectl.New(client, deployConfig, log)
				if err != nil {
					return fmt.Errorf("Error deploying devspace: deployment %s error: %v", *deployConfig.Name, err)
				}
			} else if deployConfig.Helm != nil {
				log.Info("Deploying " + *deployConfig.Name + " with helm")

				deployClient, err = helm.New(client, deployConfig, useDevOverwrite, log)
				if err != nil {
					return fmt.Errorf("Error deploying devspace: deployment %s error: %v", *deployConfig.Name, err)
				}
			} else {
				return fmt.Errorf("Error deploying devspace: deployment %s has no deployment method", *deployConfig.Name)
			}

			err = deployClient.Deploy(generatedConfig, forceDeploy)
			if err != nil {
				return fmt.Errorf("Error deploying %s: %v", *deployConfig.Name, err)
			}

			log.Donef("Finished deploying %s", *deployConfig.Name)
		}
	}

	return nil
}
