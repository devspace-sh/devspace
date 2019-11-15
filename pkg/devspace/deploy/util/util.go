package deploy

import (
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/helm"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/hook"
	kubectlpkg "github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// All deploys all deployments in the config
func All(config *latest.Config, cache *generated.CacheConfig, client kubectlpkg.Client, isDev, forceDeploy bool, builtImages map[string]string, deployments []string, log log.Logger) error {
	if config.Deployments != nil && len(config.Deployments) > 0 {
		executer := hook.NewExecuter(config, log)
		// Execute before deployments deploy hook
		err := executer.Execute(hook.Before, hook.StageDeployments, hook.All)
		if err != nil {
			return err
		}

		for _, deployConfig := range config.Deployments {
			if len(deployments) > 0 {
				shouldSkip := true

				for _, deployment := range deployments {
					if deployment == strings.TrimSpace(deployConfig.Name) {
						shouldSkip = false
						break
					}
				}

				if shouldSkip {
					continue
				}
			}

			var (
				deployClient deploy.Interface
				err          error
				method       string
			)

			if deployConfig.Kubectl != nil {
				deployClient, err = kubectl.New(config, client, deployConfig, log)
				if err != nil {
					return errors.Errorf("Error deploying devspace: deployment %s error: %v", deployConfig.Name, err)
				}

				method = "kubectl"
			} else if deployConfig.Helm != nil {
				deployClient, err = helm.New(config, client, deployConfig, log)
				if err != nil {
					return errors.Errorf("Error deploying devspace: deployment %s error: %v", deployConfig.Name, err)
				}

				method = "helm"
			} else {
				return errors.Errorf("Error deploying devspace: deployment %s has no deployment method", deployConfig.Name)
			}

			// Execute before deploment deploy hook
			err = executer.Execute(hook.Before, hook.StageDeployments, deployConfig.Name)
			if err != nil {
				return err
			}

			wasDeployed, err := deployClient.Deploy(cache, forceDeploy, builtImages)
			if err != nil {
				return errors.Errorf("Error deploying %s: %v", deployConfig.Name, err)
			}

			if wasDeployed {
				log.Donef("Successfully deployed %s with %s", deployConfig.Name, method)

				// Execute after deploment deploy hook
				err = executer.Execute(hook.After, hook.StageDeployments, deployConfig.Name)
				if err != nil {
					return err
				}
			} else {
				log.Infof("Skipping deployment %s", deployConfig.Name)
			}
		}

		// Execute after deployments deploy hook
		err = executer.Execute(hook.After, hook.StageDeployments, hook.All)
		if err != nil {
			return err
		}
	}

	return nil
}

// PurgeDeployments removes all deployments or a set of deployments from the cluster
func PurgeDeployments(config *latest.Config, cache *generated.CacheConfig, client kubectlpkg.Client, deployments []string, log log.Logger) {
	if deployments != nil && len(deployments) == 0 {
		deployments = nil
	}

	if config.Deployments != nil {
		// Reverse them
		for i := len(config.Deployments) - 1; i >= 0; i-- {
			var (
				err          error
				deployClient deploy.Interface
				deployConfig = (config.Deployments)[i]
			)

			// Check if we should skip deleting deployment
			if deployments != nil {
				found := false

				for _, value := range deployments {
					if value == deployConfig.Name {
						found = true
						break
					}
				}

				if found == false {
					continue
				}
			}

			// Delete kubectl engine
			if deployConfig.Kubectl != nil {
				deployClient, err = kubectl.New(config, client, deployConfig, log)
				if err != nil {
					log.Warnf("Unable to create kubectl deploy config: %v", err)
					continue
				}
			} else if deployConfig.Helm != nil {
				deployClient, err = helm.New(config, client, deployConfig, log)
				if err != nil {
					log.Warnf("Unable to create helm deploy config: %v", err)
					continue
				}
			}

			log.StartWait("Deleting deployment " + deployConfig.Name)
			err = deployClient.Delete(cache)
			log.StopWait()
			if err != nil {
				log.Warnf("Error deleting deployment %s: %v", deployConfig.Name, err)
			}

			log.Donef("Successfully deleted deployment %s", deployConfig.Name)
		}
	}
}
