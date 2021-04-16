package deploy

import (
	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"io"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl"
	helmclient "github.com/loft-sh/devspace/pkg/devspace/helm"
	helmtypes "github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	kubectlclient "github.com/loft-sh/devspace/pkg/devspace/kubectl"
	kubectlpkg "github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"

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
	Render(options *Options, out io.Writer, log log.Logger) error
	Purge(deployments []string, log log.Logger) error
}

type controller struct {
	config       config2.Config
	dependencies []types.Dependency

	hookExecuter hook.Executer
	client       kubectlclient.Client
}

// NewController creates a new image build controller
func NewController(config config2.Config, dependencies []types.Dependency, client kubectlclient.Client) Controller {
	config = config2.Ensure(config)
	return &controller{
		config:       config,
		dependencies: dependencies,

		hookExecuter: hook.NewExecuter(config, dependencies),
		client:       client,
	}
}

func (c *controller) Render(options *Options, out io.Writer, log log.Logger) error {
	config := c.config.Config()
	if config.Deployments != nil && len(config.Deployments) > 0 {
		helmV2Clients := map[string]helmtypes.Client{}

		for _, deployConfig := range config.Deployments {
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
			)

			if deployConfig.Kubectl != nil {
				deployClient, err = kubectl.New(c.config, c.dependencies, c.client, deployConfig, log)
				if err != nil {
					return errors.Errorf("Error render: deployment %s error: %v", deployConfig.Name, err)
				}

			} else if deployConfig.Helm != nil {
				// Get helm client
				helmClient, err := GetCachedHelmClient(c.config.Config(), deployConfig, c.client, helmV2Clients, true, log)
				if err != nil {
					return errors.Wrap(err, "get cached helm client")
				}

				deployClient, err = helm.New(c.config, c.dependencies, helmClient, c.client, deployConfig, log)
				if err != nil {
					return errors.Errorf("Error render: deployment %s error: %v", deployConfig.Name, err)
				}
			} else {
				return errors.Errorf("Error render: deployment %s has no deployment method", deployConfig.Name)
			}

			err = deployClient.Render(options.BuiltImages, out)
			if err != nil {
				return errors.Errorf("Error deploying %s: %v", deployConfig.Name, err)
			}
		}
	}

	return nil
}

// Deploy deploys all deployments in the config
func (c *controller) Deploy(options *Options, log log.Logger) error {
	config := c.config.Config()
	if config.Deployments != nil && len(config.Deployments) > 0 {
		helmV2Clients := map[string]helmtypes.Client{}

		// Execute before deployments deploy hook
		err := c.hookExecuter.Execute(hook.Before, hook.StageDeployments, hook.All, hook.Context{Client: c.client}, log)
		if err != nil {
			return err
		}

		for _, deployConfig := range config.Deployments {
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
				deployClient, err = kubectl.New(c.config, c.dependencies, c.client, deployConfig, log)
				if err != nil {
					return errors.Errorf("Error deploying: deployment %s error: %v", deployConfig.Name, err)
				}

				method = "kubectl"
			} else if deployConfig.Helm != nil {
				// Get helm client
				helmClient, err := GetCachedHelmClient(c.config.Config(), deployConfig, c.client, helmV2Clients, false, log)
				if err != nil {
					return err
				}

				deployClient, err = helm.New(c.config, c.dependencies, helmClient, c.client, deployConfig, log)
				if err != nil {
					return errors.Errorf("Error deploying: deployment %s error: %v", deployConfig.Name, err)
				}

				method = "helm"
			} else {
				return errors.Errorf("Error deploying: deployment %s has no deployment method", deployConfig.Name)
			}

			// Execute before deployment deploy hook
			err = c.hookExecuter.Execute(hook.Before, hook.StageDeployments, deployConfig.Name, hook.Context{Client: c.client}, log)
			if err != nil {
				return err
			}

			wasDeployed, err := deployClient.Deploy(options.ForceDeploy, options.BuiltImages)
			if err != nil {
				c.hookExecuter.OnError(hook.StageDeployments, []string{hook.All, deployConfig.Name}, hook.Context{Client: c.client, Error: err}, log)
				return errors.Errorf("Error deploying %s: %v", deployConfig.Name, err)
			}

			if wasDeployed {
				log.Donef("Successfully deployed %s with %s", deployConfig.Name, method)

				// Execute after deployment deploy hook
				err = c.hookExecuter.Execute(hook.After, hook.StageDeployments, deployConfig.Name, hook.Context{Client: c.client}, log)
				if err != nil {
					return err
				}
			} else {
				log.Infof("Skipping deployment %s", deployConfig.Name)
			}
		}

		// Execute after deployments deploy hook
		err = c.hookExecuter.Execute(hook.After, hook.StageDeployments, hook.All, hook.Context{Client: c.client}, log)
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

	config := c.config.Config()
	if config.Deployments != nil {
		helmV2Clients := map[string]helmtypes.Client{}

		// Execute before deployments purge hook
		err := c.hookExecuter.Execute(hook.Before, hook.StagePurgeDeployments, hook.All, hook.Context{Client: c.client}, log)
		if err != nil {
			return err
		}

		// Reverse them
		for i := len(config.Deployments) - 1; i >= 0; i-- {
			var (
				err          error
				deployClient deployer.Interface
				deployConfig = config.Deployments[i]
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
				deployClient, err = kubectl.New(c.config, c.dependencies, c.client, deployConfig, log)
				if err != nil {
					return errors.Wrap(err, "create kube client")
				}
			} else if deployConfig.Helm != nil {
				helmClient, err := GetCachedHelmClient(c.config.Config(), deployConfig, c.client, helmV2Clients, false, log)
				if err != nil {
					return errors.Wrap(err, "get cached helm client")
				}

				deployClient, err = helm.New(c.config, c.dependencies, helmClient, c.client, deployConfig, log)
				if err != nil {
					return errors.Wrap(err, "create helm client")
				}
			} else {
				return errors.Errorf("Error purging: deployment %s has no deployment method", deployConfig.Name)
			}

			// Execute before deployment purge hook
			err = c.hookExecuter.Execute(hook.Before, hook.StagePurgeDeployments, deployConfig.Name, hook.Context{Client: c.client}, log)
			if err != nil {
				return err
			}

			log.StartWait("Deleting deployment " + deployConfig.Name)
			err = deployClient.Delete()
			log.StopWait()
			if err != nil {
				// Execute on error deployment purge hook
				err = c.hookExecuter.Execute(hook.OnError, hook.StagePurgeDeployments, deployConfig.Name, hook.Context{Client: c.client}, log)
				if err != nil {
					return err
				}

				log.Warnf("Error deleting deployment %s: %v", deployConfig.Name, err)
			} else {
				// Execute after deployment purge hook
				err = c.hookExecuter.Execute(hook.After, hook.StagePurgeDeployments, deployConfig.Name, hook.Context{Client: c.client}, log)
				if err != nil {
					return err
				}
			}

			log.Donef("Successfully deleted deployment %s", deployConfig.Name)
		}

		// Execute after deployments purge hook
		err = c.hookExecuter.Execute(hook.After, hook.StagePurgeDeployments, hook.All, hook.Context{Client: c.client}, log)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetCachedHelmClient returns a helm client that could be cached in a helmV2Clients map. If not found it will add it to the map and create it
func GetCachedHelmClient(config *latest.Config, deployConfig *latest.DeploymentConfig, client kubectlpkg.Client, helmV2Clients map[string]helmtypes.Client, dryInit bool, log log.Logger) (helmtypes.Client, error) {
	var (
		err        error
		helmClient helmtypes.Client
	)

	tillerNamespace := getTillerNamespace(client, deployConfig)
	if tillerNamespace != "" && helmV2Clients[tillerNamespace] != nil {
		helmClient = helmV2Clients[tillerNamespace]
	} else {
		helmClient, err = helmclient.NewClient(config, deployConfig, client, tillerNamespace, false, dryInit, log)
		if err != nil {
			return nil, err
		}

		if tillerNamespace != "" {
			helmV2Clients[tillerNamespace] = helmClient
		}
	}

	return helmClient, nil
}

func getTillerNamespace(kubeClient kubectlpkg.Client, deployConfig *latest.DeploymentConfig) string {
	if kubeClient != nil && deployConfig.Helm != nil && deployConfig.Helm.V2 == true {
		tillerNamespace := kubeClient.Namespace()
		if deployConfig.Helm.TillerNamespace != "" {
			tillerNamespace = deployConfig.Helm.TillerNamespace
		}

		return tillerNamespace
	}

	return ""
}
