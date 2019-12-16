package deploy

import (
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/helm"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/kubectl"
	helmclient "github.com/devspace-cloud/devspace/pkg/devspace/helm"
	helmtypes "github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/hook"
	kubectlclient "github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	kubectlpkg "github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// Options describe how the deployments should be deployed
type Options struct {
	IsDev       bool
	ForceDeploy bool
	BuiltImages map[string]string
	Deployments []string
}

// Controller is the main deploying interface
type Controller interface {
	Deploy(options *Options, log log.Logger) error
	Purge(deployments []string, log log.Logger) error
}

type controller struct {
	config *latest.Config
	cache  *generated.CacheConfig

	hookExecuter hook.Executer
	client       kubectlclient.Client
}

// NewController creates a new image build controller
func NewController(config *latest.Config, cache *generated.CacheConfig, client kubectlclient.Client) Controller {
	return &controller{
		config: config,
		cache:  cache,

		hookExecuter: hook.NewExecuter(config),
		client:       client,
	}
}

// DeployAll deploys all deployments in the config
func (c *controller) Deploy(options *Options, log log.Logger) error {
	if c.config.Deployments != nil && len(c.config.Deployments) > 0 {
		helmV2Clients := map[string]helmtypes.Client{}

		// Execute before deployments deploy hook
		err := c.hookExecuter.Execute(hook.Before, hook.StageDeployments, hook.All, log)
		if err != nil {
			return err
		}

		for _, deployConfig := range c.config.Deployments {
			if len(options.Deployments) > 0 {
				shouldSkip := true

				for _, deployment := range options.Deployments {
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
				deployClient deployer.Interface
				err          error
				method       string
			)

			if deployConfig.Kubectl != nil {
				deployClient, err = kubectl.New(c.config, c.client, deployConfig, log)
				if err != nil {
					return errors.Errorf("Error deploying devspace: deployment %s error: %v", deployConfig.Name, err)
				}

				method = "kubectl"
			} else if deployConfig.Helm != nil {
				// Get helm client
				helmClient, err := GetCachedHelmClient(c.config, deployConfig, c.client, helmV2Clients, log)
				if err != nil {
					return err
				}

				deployClient, err = helm.New(c.config, helmClient, c.client, deployConfig, log)
				if err != nil {
					return errors.Errorf("Error deploying devspace: deployment %s error: %v", deployConfig.Name, err)
				}

				method = "helm"
			} else {
				return errors.Errorf("Error deploying devspace: deployment %s has no deployment method", deployConfig.Name)
			}

			// Execute before deploment deploy hook
			err = c.hookExecuter.Execute(hook.Before, hook.StageDeployments, deployConfig.Name, log)
			if err != nil {
				return err
			}

			wasDeployed, err := deployClient.Deploy(c.cache, options.ForceDeploy, options.BuiltImages)
			if err != nil {
				return errors.Errorf("Error deploying %s: %v", deployConfig.Name, err)
			}

			if wasDeployed {
				log.Donef("Successfully deployed %s with %s", deployConfig.Name, method)

				// Execute after deploment deploy hook
				err = c.hookExecuter.Execute(hook.After, hook.StageDeployments, deployConfig.Name, log)
				if err != nil {
					return err
				}
			} else {
				log.Infof("Skipping deployment %s", deployConfig.Name)
			}
		}

		// Execute after deployments deploy hook
		err = c.hookExecuter.Execute(hook.After, hook.StageDeployments, hook.All, log)
		if err != nil {
			return err
		}
	}

	return nil
}

// Purge removes all deployments or a set of deployments from the cluster
func (c *controller) Purge(deployments []string, log log.Logger) error {
	if deployments != nil && len(deployments) == 0 {
		deployments = nil
	}

	if c.config.Deployments != nil {
		helmV2Clients := map[string]helmtypes.Client{}

		// Reverse them
		for i := len(c.config.Deployments) - 1; i >= 0; i-- {
			var (
				err          error
				deployClient deployer.Interface
				deployConfig = (c.config.Deployments)[i]
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
				deployClient, err = kubectl.New(c.config, c.client, deployConfig, log)
				if err != nil {
					return errors.Wrap(err, "create kube client")
				}
			} else if deployConfig.Helm != nil {
				helmClient, err := GetCachedHelmClient(c.config, deployConfig, c.client, helmV2Clients, log)
				if err != nil {
					return errors.Wrap(err, "get cached helm client")
				}

				deployClient, err = helm.New(c.config, helmClient, c.client, deployConfig, log)
				if err != nil {
					return errors.Wrap(err, "create helm client")
				}
			}

			log.StartWait("Deleting deployment " + deployConfig.Name)
			err = deployClient.Delete(c.cache)
			log.StopWait()
			if err != nil {
				log.Warnf("Error deleting deployment %s: %v", deployConfig.Name, err)
			}

			log.Donef("Successfully deleted deployment %s", deployConfig.Name)
		}
	}

	return nil
}

// GetCachedHelmClient returns a helm client that could be cached in a helmV2Clients map. If not found it will add it to the map and create it
func GetCachedHelmClient(config *latest.Config, deployConfig *latest.DeploymentConfig, client kubectlpkg.Client, helmV2Clients map[string]helmtypes.Client, log log.Logger) (helmtypes.Client, error) {
	var (
		err        error
		helmClient helmtypes.Client
	)

	tillerNamespace := getTillernamespace(client, deployConfig)
	if tillerNamespace != "" && helmV2Clients[tillerNamespace] != nil {
		helmClient = helmV2Clients[tillerNamespace]
	} else {
		helmClient, err = helmclient.NewClient(config, deployConfig, client, tillerNamespace, false, log)
		if err != nil {
			return nil, err
		}

		if tillerNamespace != "" {
			helmV2Clients[tillerNamespace] = helmClient
		}
	}

	return helmClient, nil
}

func getTillernamespace(kubeClient kubectlpkg.Client, deployConfig *latest.DeploymentConfig) string {
	if deployConfig.Helm != nil && deployConfig.Helm.V2 == true {
		tillerNamespace := kubeClient.Namespace()
		if deployConfig.Helm.TillerNamespace != "" {
			tillerNamespace = deployConfig.Helm.TillerNamespace
		}

		return tillerNamespace
	}

	return ""
}
