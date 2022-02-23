package deploy

import (
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"io"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl"
	helmclient "github.com/loft-sh/devspace/pkg/devspace/helm"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/scanner"

	"github.com/pkg/errors"
)

// Options describe how the deployments should be deployed
type Options struct {
	ForceDeploy bool
}

// Controller is the main deploying interface
type Controller interface {
	Deploy(ctx *devspacecontext.Context, deployments []string, options *Options) error
	Render(ctx *devspacecontext.Context, deployments []string, options *Options, out io.Writer) error
	Purge(ctx *devspacecontext.Context, deployments []string) error
}

type controller struct{}

// NewController creates a new image build controller
func NewController() Controller {
	return &controller{}
}

func (c *controller) Render(ctx *devspacecontext.Context, deployments []string, options *Options, out io.Writer) error {
	config := ctx.Config.Config()
	if config.Deployments != nil && len(config.Deployments) > 0 {
		// Execute before deployments deploy hook
		err := hook.ExecuteHooks(ctx, nil, "before:render")
		if err != nil {
			return err
		}

		for _, deployConfig := range config.Deployments {
			if deployConfig.Disabled {
				ctx.Log.Debugf("Skip deployment %s, because it is disabled", deployConfig.Name)
				continue
			}

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

			deployClient, err := c.getDeployClient(ctx, deployConfig)
			if err != nil {
				return err
			}

			hookErr := hook.ExecuteHooks(ctx, map[string]interface{}{
				"DEPLOY_NAME":   deployConfig.Name,
				"DEPLOY_CONFIG": deployConfig,
			}, hook.EventsForSingle("before:render", deployConfig.Name).With("deploy.beforeRender")...)
			if hookErr != nil {
				return hookErr
			}

			err = deployClient.Render(ctx, out)
			if err != nil {
				hookErr := hook.ExecuteHooks(ctx, map[string]interface{}{
					"DEPLOY_NAME":   deployConfig.Name,
					"DEPLOY_CONFIG": deployConfig,
					"ERROR":         err,
				}, hook.EventsForSingle("error:render", deployConfig.Name).With("deploy.errorRender")...)
				if hookErr != nil {
					return hookErr
				}

				return errors.Errorf("error deploying %s: %v", deployConfig.Name, err)
			}

			hookErr = hook.ExecuteHooks(ctx, map[string]interface{}{
				"DEPLOY_NAME":   deployConfig.Name,
				"DEPLOY_CONFIG": deployConfig,
			}, hook.EventsForSingle("after:render", deployConfig.Name).With("deploy.afterRender")...)
			if hookErr != nil {
				return hookErr
			}
		}

		err = hook.ExecuteHooks(ctx, nil, "after:render")
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) getDeployClient(ctx *devspacecontext.Context, deployConfig *latest.DeploymentConfig) (deployer.Interface, error) {
	var (
		deployClient deployer.Interface
		err          error
	)
	if deployConfig.Kubectl != nil {
		deployClient, err = kubectl.New(ctx, deployConfig)
		if err != nil {
			return nil, errors.Errorf("error render: deployment %s error: %v", deployConfig.Name, err)
		}
	} else if deployConfig.Helm != nil {
		// Get helm client
		helmClient, err := helmclient.NewClient(ctx.Log)
		if err != nil {
			return nil, err
		}

		deployClient, err = helm.New(ctx, helmClient, deployConfig)
		if err != nil {
			return nil, errors.Errorf("error render: deployment %s error: %v", deployConfig.Name, err)
		}
	} else {
		return nil, errors.Errorf("error render: deployment %s has no deployment method", deployConfig.Name)
	}
	return deployClient, nil
}

// Deploy deploys all deployments in the config
func (c *controller) Deploy(ctx *devspacecontext.Context, deployments []string, options *Options) error {
	config := ctx.Config.Config()
	if config.Deployments != nil && len(config.Deployments) > 0 {
		// Execute before deployments deploy hook
		err := hook.ExecuteHooks(ctx, nil, "before:deploy")
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
				logsLog := log.NewPrefixLogger("["+deployConfig.Name+"] ", log.Colors[(len(log.Colors)-1)-(deployNumber%len(log.Colors))], ctx.Log)
				go func() {
					scanner := scanner.NewScanner(reader)
					for scanner.Scan() {
						logsLog.Info(scanner.Text())
					}
				}()

				wasDeployed, err := c.deployOne(ctx.WithLogger(streamLog), deployConfig, deployments, options)
				_ = writer.Close()
				if err != nil {
					errChan <- err
				} else {
					deployedChan <- wasDeployed
				}
			}(deployConfig, i)
		}

		if len(concurrentDeployments) > 0 {
			ctx.Log.StartWait(fmt.Sprintf("Deploying %d deployments concurrently", len(concurrentDeployments)))

			// Wait for concurrent deployments to complete before starting sequential deployments.
			for i := 0; i < len(concurrentDeployments); i++ {
				select {
				case err := <-errChan:
					return err
				case <-deployedChan:
					ctx.Log.StartWait(fmt.Sprintf("Deploying %d deployments concurrently", len(concurrentDeployments)-i-1))

				}
			}

			ctx.Log.StopWait()
		}

		for _, deployConfig := range sequentialDeployments {
			_, err := c.deployOne(ctx, deployConfig, deployments, options)
			if err != nil {
				return err
			}
		}

		// Execute after deployments deploy hook
		err = hook.ExecuteHooks(ctx, nil, "after:deploy")
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) deployOne(ctx *devspacecontext.Context, deployConfig *latest.DeploymentConfig, deployments []string, options *Options) (bool, error) {
	if deployConfig.Disabled {
		ctx.Log.Debugf("Skip deployment %s, because it is disabled", deployConfig.Name)
		return true, nil
	}

	if len(deployments) > 0 {
		shouldSkip := true
		for _, deployment := range deployments {
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

	if deployConfig.Namespace != "" {
		err = ctx.KubeClient.EnsureNamespace(ctx.Context, deployConfig.Namespace, ctx.Log)
		if err != nil {
			return false, err
		}
	}

	if deployConfig.Kubectl != nil {
		deployClient, err = kubectl.New(ctx, deployConfig)
		if err != nil {
			return true, errors.Errorf("error deploying: deployment %s error: %v", deployConfig.Name, err)
		}

		method = "kubectl"
	} else if deployConfig.Helm != nil {
		// Get helm client
		helmClient, err := helmclient.NewClient(ctx.Log)
		if err != nil {
			return true, err
		}

		deployClient, err = helm.New(ctx, helmClient, deployConfig)
		if err != nil {
			return true, errors.Errorf("error deploying: deployment %s error: %v", deployConfig.Name, err)
		}

		method = "helm"
	} else {
		return true, errors.Errorf("error deploying: deployment %s has no deployment method", deployConfig.Name)
	}
	// Execute before deployment deploy hook
	err = hook.ExecuteHooks(ctx, map[string]interface{}{
		"DEPLOY_NAME":   deployConfig.Name,
		"DEPLOY_CONFIG": deployConfig,
	}, hook.EventsForSingle("before:deploy", deployConfig.Name).With("deploy.beforeDeploy")...)
	if err != nil {
		return true, err
	}

	wasDeployed, err := deployClient.Deploy(ctx, options.ForceDeploy)
	if err != nil {
		hookErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"DEPLOY_NAME":   deployConfig.Name,
			"DEPLOY_CONFIG": deployConfig,
			"ERROR":         err,
		}, hook.EventsForSingle("error:deploy", deployConfig.Name).With("deploy.errorDeploy")...)
		if hookErr != nil {
			return true, hookErr
		}

		return true, errors.Errorf("error deploying %s: %v", deployConfig.Name, err)
	}

	if wasDeployed {
		ctx.Log.Donef("Successfully deployed %s with %s", deployConfig.Name, method)
		// Execute after deployment deploy hook
		err = hook.ExecuteHooks(ctx, map[string]interface{}{
			"DEPLOY_NAME":   deployConfig.Name,
			"DEPLOY_CONFIG": deployConfig,
		}, hook.EventsForSingle("after:deploy", deployConfig.Name).With("deploy.afterDeploy")...)
		if err != nil {
			return true, err
		}
	} else {
		ctx.Log.Infof("Skipping deployment %s", deployConfig.Name)
		// Execute skip deploy hook
		err = hook.ExecuteHooks(ctx, map[string]interface{}{
			"DEPLOY_NAME":   deployConfig.Name,
			"DEPLOY_CONFIG": deployConfig,
		}, hook.EventsForSingle("skip:deploy", deployConfig.Name)...)
		if err != nil {
			return true, err
		}
	}
	return false, nil
}

// Purge removes all deployments or a set of deployments from the cluster
func (c *controller) Purge(ctx *devspacecontext.Context, deployments []string) error {
	if deployments != nil && len(deployments) == 0 {
		deployments = nil
	}

	config := ctx.Config.Config()
	if config.Deployments != nil {
		// Execute before deployments purge hook
		err := hook.ExecuteHooks(ctx, nil, "before:purge")
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
				ctx.Log.Debugf("Skip deployment %s, because it is disabled", deployConfig.Name)
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
				deployClient, err = kubectl.New(ctx, deployConfig)
				if err != nil {
					return errors.Wrap(err, "create kube client")
				}
			} else if deployConfig.Helm != nil {
				helmClient, err := helmclient.NewClient(ctx.Log)
				if err != nil {
					return errors.Wrap(err, "get cached helm client")
				}

				deployClient, err = helm.New(ctx, helmClient, deployConfig)
				if err != nil {
					return errors.Wrap(err, "create helm client")
				}
			} else {
				return errors.Errorf("error purging: deployment %s has no deployment method", deployConfig.Name)
			}

			// Execute before deployment purge hook
			err = hook.ExecuteHooks(ctx, map[string]interface{}{
				"DEPLOY_NAME":   deployConfig.Name,
				"DEPLOY_CONFIG": deployConfig,
			}, hook.EventsForSingle("before:purge", deployConfig.Name).With("deploy.beforePurge")...)
			if err != nil {
				return err
			}

			ctx.Log.StartWait("Deleting deployment " + deployConfig.Name)
			err = deployClient.Delete(ctx)
			ctx.Log.StopWait()
			if err != nil {
				// Execute on error deployment purge hook
				hookErr := hook.ExecuteHooks(ctx, map[string]interface{}{
					"DEPLOY_NAME":   deployConfig.Name,
					"DEPLOY_CONFIG": deployConfig,
					"ERROR":         err,
				}, hook.EventsForSingle("error:purge", deployConfig.Name).With("deploy.errorPurge")...)
				if hookErr != nil {
					return hookErr
				}

				ctx.Log.Warnf("Error deleting deployment %s: %v", deployConfig.Name, err)
			} else {
				err = hook.ExecuteHooks(ctx, map[string]interface{}{
					"DEPLOY_NAME":   deployConfig.Name,
					"DEPLOY_CONFIG": deployConfig,
				}, hook.EventsForSingle("after:purge", deployConfig.Name).With("deploy.afterPurge")...)
				if err != nil {
					return err
				}
			}

			ctx.Log.Donef("Successfully deleted deployment %s", deployConfig.Name)
		}

		// Execute after deployments purge hook
		err = hook.ExecuteHooks(ctx, nil, "after:purge")
		if err != nil {
			return err
		}
	}

	return nil
}
