package deploy

import (
	"fmt"
	"io"
	"strings"

	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl"
	helmclient "github.com/loft-sh/devspace/pkg/devspace/helm"
	helmtypes "github.com/loft-sh/devspace/pkg/devspace/helm/types"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	kubectlclient "github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/scanner"

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
	client       kubectlclient.Client
}

// NewController creates a new image build controller
func NewController(config config2.Config, dependencies []types.Dependency, client kubectlclient.Client) Controller {
	config = config2.Ensure(config)
	return &controller{
		config:       config,
		dependencies: dependencies,
		client:       client,
	}
}

func (c *controller) Render(options *Options, out io.Writer, log log.Logger) error {
	config := c.config.Config()
	if config.Deployments != nil && len(config.Deployments) > 0 {
		helmV2Clients := map[string]helmtypes.Client{}

		// Execute before deployments deploy hook
		err := hook.ExecuteHooks(c.client, c.config, c.dependencies, nil, log, "before:render")
		if err != nil {
			return err
		}

		for _, deployConfig := range config.Deployments {
			if deployConfig.Disabled {
				log.Debugf("Skip deployment %s, because it is disabled", deployConfig.Name)
				continue
			}

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

			deployClient, err := c.getDeployClient(deployConfig, helmV2Clients, log)
			if err != nil {
				return err
			}

			hookErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
				"DEPLOY_NAME":   deployConfig.Name,
				"DEPLOY_CONFIG": deployConfig,
			}, log, hook.EventsForSingle("before:render", deployConfig.Name).With("deploy.beforeRender")...)
			if hookErr != nil {
				return hookErr
			}

			err = deployClient.Render(options.BuiltImages, out)
			if err != nil {
				hookErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
					"DEPLOY_NAME":   deployConfig.Name,
					"DEPLOY_CONFIG": deployConfig,
					"ERROR":         err,
				}, log, hook.EventsForSingle("error:render", deployConfig.Name).With("deploy.errorRender")...)
				if hookErr != nil {
					return hookErr
				}

				return errors.Errorf("error deploying %s: %v", deployConfig.Name, err)
			}

			hookErr = hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
				"DEPLOY_NAME":   deployConfig.Name,
				"DEPLOY_CONFIG": deployConfig,
			}, log, hook.EventsForSingle("after:render", deployConfig.Name).With("deploy.afterRender")...)
			if hookErr != nil {
				return hookErr
			}
		}

		err = hook.ExecuteHooks(c.client, c.config, c.dependencies, nil, log, "after:render")
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) getDeployClient(deployConfig *latest.DeploymentConfig, helmV2Clients map[string]helmtypes.Client, log log.Logger) (deployer.Interface, error) {
	var (
		deployClient deployer.Interface
		err          error
	)
	if deployConfig.Kubectl != nil {
		deployClient, err = kubectl.New(c.config, c.dependencies, c.client, deployConfig, log)
		if err != nil {
			return nil, errors.Errorf("error render: deployment %s error: %v", deployConfig.Name, err)
		}

	} else if deployConfig.Helm != nil {
		// Get helm client
		helmClient, err := GetCachedHelmClient(c.config.Config(), deployConfig, c.client, helmV2Clients, true, log)
		if err != nil {
			return nil, errors.Wrap(err, "get cached helm client")
		}

		deployClient, err = helm.New(c.config, c.dependencies, helmClient, c.client, deployConfig, log)
		if err != nil {
			return nil, errors.Errorf("error render: deployment %s error: %v", deployConfig.Name, err)
		}
	} else {
		return nil, errors.Errorf("error render: deployment %s has no deployment method", deployConfig.Name)
	}
	return deployClient, nil
}

// Deploy deploys all deployments in the config
func (c *controller) Deploy(options *Options, logLogger log.Logger) error {
	config := c.config.Config()
	if config.Deployments != nil && len(config.Deployments) > 0 {
		helmV2Clients := map[string]helmtypes.Client{}

		// Execute before deployments deploy hook
		err := hook.ExecuteHooks(c.client, c.config, c.dependencies, nil, logLogger, "before:deploy")
		if err != nil {
			return err
		}

		var (
			concurrentDeployments []*latest.DeploymentConfig
			sequentialDeployments []*latest.DeploymentConfig
		)

		for _, deployConfig := range config.Deployments {
			if deployConfig.Concurrent {
				concurrentDeployments = append(concurrentDeployments, deployConfig)
			} else {
				sequentialDeployments = append(sequentialDeployments, deployConfig)
			}
		}

		var (
			errChan      = make(chan error)
			deployedChan = make(chan bool)
		)

		for i, deployConfig := range concurrentDeployments {
			go func(deployConfig *latest.DeploymentConfig, deployNumber int) {
				// Create new logger to allow concurrent logging.
				reader, writer := io.Pipe()
				streamLog := log.NewStreamLogger(writer, logrus.InfoLevel)
				logsLog := log.NewPrefixLogger("["+deployConfig.Name+"] ", log.Colors[(len(log.Colors)-1)-(deployNumber%len(log.Colors))], logLogger)
				go func() {
					scanner := scanner.NewScanner(reader)
					for scanner.Scan() {
						logsLog.Info(scanner.Text())
					}
				}()

				wasDeployed, err := c.deployOne(deployConfig, streamLog, options, helmV2Clients)
				_ = writer.Close()
				if err != nil {
					errChan <- err
				} else {
					deployedChan <- wasDeployed
				}
			}(deployConfig, i)
		}

		if len(concurrentDeployments) > 0 {
			logLogger.StartWait(fmt.Sprintf("Deploying %d deployments concurrently", len(concurrentDeployments)))

			// Wait for concurrent deployments to complete before starting sequential deployments.
			for i := 0; i < len(concurrentDeployments); i++ {
				select {
				case err := <-errChan:
					return err
				case <-deployedChan:
					logLogger.StartWait(fmt.Sprintf("Deploying %d deployments concurrently", len(concurrentDeployments)-i-1))

				}
			}

			logLogger.StopWait()
		}

		for _, deployConfig := range sequentialDeployments {
			_, err := c.deployOne(deployConfig, logLogger, options, helmV2Clients)
			if err != nil {
				return err
			}
		}

		// Execute after deployments deploy hook
		err = hook.ExecuteHooks(c.client, c.config, c.dependencies, nil, logLogger, "after:deploy")
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) deployOne(deployConfig *latest.DeploymentConfig, log log.Logger, options *Options, helmV2Clients map[string]helmtypes.Client) (bool, error) {
	if deployConfig.Disabled {
		log.Debugf("Skip deployment %s, because it is disabled", deployConfig.Name)
		return true, nil
	}

	if len(options.Deployments) > 0 {
		shouldSkip := true

		for _, deployment := range options.Deployments {
			if deployment == strings.TrimSpace(deployConfig.Name) {
				shouldSkip = false
				break
			}
		}

		if shouldSkip {
			return true, nil
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
			return true, errors.Errorf("error deploying: deployment %s error: %v", deployConfig.Name, err)
		}

		method = "kubectl"
	} else if deployConfig.Helm != nil {
		// Get helm client
		helmClient, err := GetCachedHelmClient(c.config.Config(), deployConfig, c.client, helmV2Clients, false, log)
		if err != nil {
			return true, err
		}

		deployClient, err = helm.New(c.config, c.dependencies, helmClient, c.client, deployConfig, log)
		if err != nil {
			return true, errors.Errorf("error deploying: deployment %s error: %v", deployConfig.Name, err)
		}

		method = "helm"
	} else {
		return true, errors.Errorf("error deploying: deployment %s has no deployment method", deployConfig.Name)
	}
	// Execute before deployment deploy hook
	err = hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
		"DEPLOY_NAME":   deployConfig.Name,
		"DEPLOY_CONFIG": deployConfig,
	}, log, hook.EventsForSingle("before:deploy", deployConfig.Name).With("deploy.beforeDeploy")...)
	if err != nil {
		return true, err
	}

	wasDeployed, err := deployClient.Deploy(options.ForceDeploy, options.BuiltImages)
	if err != nil {
		hookErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
			"DEPLOY_NAME":   deployConfig.Name,
			"DEPLOY_CONFIG": deployConfig,
			"ERROR":         err,
		}, log, hook.EventsForSingle("error:deploy", deployConfig.Name).With("deploy.errorDeploy")...)
		if hookErr != nil {
			return true, hookErr
		}

		return true, errors.Errorf("error deploying %s: %v", deployConfig.Name, err)
	}

	if wasDeployed {
		log.Donef("Successfully deployed %s with %s", deployConfig.Name, method)
		// Execute after deployment deploy hook
		err = hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
			"DEPLOY_NAME":   deployConfig.Name,
			"DEPLOY_CONFIG": deployConfig,
		}, log, hook.EventsForSingle("after:deploy", deployConfig.Name).With("deploy.afterDeploy")...)
		if err != nil {
			return true, err
		}
	} else {
		log.Infof("Skipping deployment %s", deployConfig.Name)
		// Execute skip deploy hook
		err = hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
			"DEPLOY_NAME":   deployConfig.Name,
			"DEPLOY_CONFIG": deployConfig,
		}, log, hook.EventsForSingle("skip:deploy", deployConfig.Name)...)
		if err != nil {
			return true, err
		}
	}
	return false, nil
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
		err := hook.ExecuteHooks(c.client, c.config, c.dependencies, nil, log, "before:purge")
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
			if deployConfig.Disabled {
				log.Debugf("Skip deployment %s, because it is disabled", deployConfig.Name)
				continue
			}

			// Check if we should skip deleting deployment
			if deployments != nil {
				found := false

				for _, value := range deployments {
					if value == deployConfig.Name {
						found = true
						break
					}
				}

				if !found {
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
				return errors.Errorf("error purging: deployment %s has no deployment method", deployConfig.Name)
			}

			// Execute before deployment purge hook
			err = hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
				"DEPLOY_NAME":   deployConfig.Name,
				"DEPLOY_CONFIG": deployConfig,
			}, log, hook.EventsForSingle("before:purge", deployConfig.Name).With("deploy.beforePurge")...)
			if err != nil {
				return err
			}

			log.StartWait("Deleting deployment " + deployConfig.Name)
			err = deployClient.Delete()
			log.StopWait()
			if err != nil {
				// Execute on error deployment purge hook
				hookErr := hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
					"DEPLOY_NAME":   deployConfig.Name,
					"DEPLOY_CONFIG": deployConfig,
					"ERROR":         err,
				}, log, hook.EventsForSingle("error:purge", deployConfig.Name).With("deploy.errorPurge")...)
				if hookErr != nil {
					return hookErr
				}

				log.Warnf("Error deleting deployment %s: %v", deployConfig.Name, err)
			} else {
				err = hook.ExecuteHooks(c.client, c.config, c.dependencies, map[string]interface{}{
					"DEPLOY_NAME":   deployConfig.Name,
					"DEPLOY_CONFIG": deployConfig,
				}, log, hook.EventsForSingle("after:purge", deployConfig.Name).With("deploy.afterPurge")...)
				if err != nil {
					return err
				}
			}

			log.Donef("Successfully deleted deployment %s", deployConfig.Name)
		}

		// Execute after deployments purge hook
		err = hook.ExecuteHooks(c.client, c.config, c.dependencies, nil, log, "after:purge")
		if err != nil {
			return err
		}
	}

	return nil
}

// GetCachedHelmClient returns a helm client that could be cached in a helmV2Clients map. If not found it will add it to the map and create it
func GetCachedHelmClient(config *latest.Config, deployConfig *latest.DeploymentConfig, client kubectlclient.Client, helmV2Clients map[string]helmtypes.Client, dryInit bool, log log.Logger) (helmtypes.Client, error) {
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

func getTillerNamespace(kubeClient kubectlclient.Client, deployConfig *latest.DeploymentConfig) string {
	if kubeClient != nil && deployConfig.Helm != nil && deployConfig.Helm.V2 {
		tillerNamespace := kubeClient.Namespace()
		if deployConfig.Helm.TillerNamespace != "" {
			tillerNamespace = deployConfig.Helm.TillerNamespace
		}

		return tillerNamespace
	}

	return ""
}
