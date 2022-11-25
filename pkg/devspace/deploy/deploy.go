package deploy

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/tanka"
	helmclient "github.com/loft-sh/devspace/pkg/devspace/helm"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	kubectlclient "github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
)

// Options describe how the deployments should be deployed
type Options struct {
	SkipDeploy  bool `long:"skip-deploy" description:"If enabled, will skip deploying"`
	ForceDeploy bool `long:"force-redeploy" description:"Forces redeployment"`
	Sequential  bool `long:"sequential" description:"Sequentially deploys the deployments"`

	Render       bool `long:"render" description:"If true, prints the rendered manifests to the stdout instead of deploying them"`
	RenderWriter io.Writer
}

type PurgeOptions struct {
	ForcePurge bool `long:"force-purge" description:"Forces purging of deployments even though they might be still in use by other DevSpace projects"`
}

// Controller is the main deploying interface
type Controller interface {
	Deploy(ctx devspacecontext.Context, deployments []string, options *Options) error
	Purge(ctx devspacecontext.Context, deployments []string, options *PurgeOptions) error
}

type controller struct{}

// NewController creates a new image build controller
func NewController() Controller {
	return &controller{}
}

// Deploy deploys all deployments in the config
func (c *controller) Deploy(ctx devspacecontext.Context, deployments []string, options *Options) error {
	config := ctx.Config().Config()
	event := "deploy"
	if options.Render {
		event = "render"
	}

	if options.SkipDeploy {
		ctx.Log().Debugf("Skip deploy because of --skip-deploy")
		return nil
	}

	if config.Deployments != nil && len(config.Deployments) > 0 {
		// Execute before deployments deploy hook
		err := hook.ExecuteHooks(ctx, nil, "before:"+event)
		if err != nil {
			return err
		}

		// get relevant deployments
		var (
			concurrentDeployments []*latest.DeploymentConfig
			sequentialDeployments []*latest.DeploymentConfig
		)
		if len(deployments) == 0 {
			for _, deployConfig := range config.Deployments {
				if !options.Render && !options.Sequential {
					concurrentDeployments = append(concurrentDeployments, deployConfig)
				} else {
					sequentialDeployments = append(sequentialDeployments, deployConfig)
				}
			}

			// make sure --all behaves the same every rung
			sort.Slice(concurrentDeployments, func(i, j int) bool {
				return concurrentDeployments[i].Name < concurrentDeployments[j].Name
			})
			sort.Slice(sequentialDeployments, func(i, j int) bool {
				return sequentialDeployments[i].Name < sequentialDeployments[j].Name
			})
		} else {
			deploymentMap := config.Deployments
			if deploymentMap == nil {
				deploymentMap = map[string]*latest.DeploymentConfig{}
			}

			for _, deployment := range deployments {
				deployConfig, ok := deploymentMap[deployment]
				if !ok {
					return fmt.Errorf("couldn't find deployment %v", deployment)
				}

				if !options.Render && !options.Sequential {
					concurrentDeployments = append(concurrentDeployments, deployConfig)
				} else {
					sequentialDeployments = append(sequentialDeployments, deployConfig)
				}
			}
		}

		var (
			errChan      = make(chan error)
			deployedChan = make(chan bool)
		)
		for i, deployConfig := range concurrentDeployments {
			go func(deployConfig *latest.DeploymentConfig, deployNumber int) {
				wasDeployed, err := c.deployOne(ctx.WithLogger(ctx.Log().WithPrefix("deploy:"+deployConfig.Name+" ")), deployConfig, options)
				if err != nil {
					errChan <- err
				} else {
					deployedChan <- wasDeployed
				}
			}(deployConfig, i)
		}

		if len(concurrentDeployments) > 0 {
			ctx.Log().Debugf("Deploying %d deployments concurrently...", len(concurrentDeployments))

			// Wait for concurrent deployments to complete before starting sequential deployments.
			for i := 0; i < len(concurrentDeployments); i++ {
				select {
				case err := <-errChan:
					return err
				case <-deployedChan:
					ctx.Log().Debugf("Deploying %d deployments concurrently", len(concurrentDeployments)-i-1)
				}
			}
		}

		for _, deployConfig := range sequentialDeployments {
			logsDeploy := ctx.Log().WithPrefix("deploy:" + deployConfig.Name + " ")
			_, err := c.deployOne(ctx.WithLogger(logsDeploy), deployConfig, options)
			if err != nil {
				return err
			}
		}

		err = ctx.Config().RemoteCache().Save(ctx.Context(), ctx.KubeClient())
		if err != nil {
			return err
		}

		// Execute after deployments deploy hook
		err = hook.ExecuteHooks(ctx, nil, "after:"+event)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) deployOne(ctx devspacecontext.Context, deployConfig *latest.DeploymentConfig, options *Options) (bool, error) {
	event := "deploy"
	if options.Render {
		event = "render"
	}

	var (
		deployClient deployer.Interface
		err          error
		method       string
	)

	if !options.Render && deployConfig.Namespace != "" {
		err = kubectlclient.EnsureNamespace(ctx.Context(), ctx.KubeClient(), deployConfig.Namespace, ctx.Log())
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
		helmClient, err := helmclient.NewClient(ctx.Log())
		if err != nil {
			return true, err
		}

		deployClient, err = helm.New(helmClient, deployConfig)
		if err != nil {
			return true, errors.Errorf("error deploying: deployment %s error: %v", deployConfig.Name, err)
		}

		method = "helm"
	} else if deployConfig.Tanka != nil {
		deployClient, err = tanka.New(ctx, deployConfig)
		if err != nil {
			return true, errors.Errorf("error deploying: deployment %s error: %v", deployConfig.Name, err)
		}
		method = "tanka"

	} else {
		return true, errors.Errorf("error deploying: deployment %s has no deployment method", deployConfig.Name)
	}
	// Execute before deployment deploy hook
	err = hook.ExecuteHooks(ctx, map[string]interface{}{
		"DEPLOY_NAME":   deployConfig.Name,
		"DEPLOY_CONFIG": deployConfig,
	}, hook.EventsForSingle("before:"+event, deployConfig.Name)...)
	if err != nil {
		return true, err
	}

	wasDeployed := false
	if !options.Render {
		wasDeployed, err = deployClient.Deploy(ctx, options.ForceDeploy)
	} else {
		err = deployClient.Render(ctx, options.RenderWriter)
	}
	if err != nil {
		hookErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"DEPLOY_NAME":   deployConfig.Name,
			"DEPLOY_CONFIG": deployConfig,
			"ERROR":         err,
		}, hook.EventsForSingle("error:"+event, deployConfig.Name)...)
		if hookErr != nil {
			return true, hookErr
		}

		return true, errors.Errorf("error deploying %s: %v", deployConfig.Name, err)
	}

	if wasDeployed {
		ctx.Log().Donef("Successfully deployed %s with %s", ansi.Color(deployConfig.Name, "white+b"), ansi.Color(method, "white+b"))
		// Execute after deployment deploy hook
		err = hook.ExecuteHooks(ctx, map[string]interface{}{
			"DEPLOY_NAME":   deployConfig.Name,
			"DEPLOY_CONFIG": deployConfig,
		}, hook.EventsForSingle("after:"+event, deployConfig.Name)...)
		if err != nil {
			return true, err
		}
	} else if !options.Render {
		ctx.Log().Infof("Skipping deployment %s", deployConfig.Name)
		// Execute skip deploy hook
		err = hook.ExecuteHooks(ctx, map[string]interface{}{
			"DEPLOY_NAME":   deployConfig.Name,
			"DEPLOY_CONFIG": deployConfig,
		}, hook.EventsForSingle("skip:"+event, deployConfig.Name)...)
		if err != nil {
			return true, err
		}
	}
	return false, nil
}

// Purge removes all deployments or a set of deployments from the cluster
func (c *controller) Purge(ctx devspacecontext.Context, deployments []string, options *PurgeOptions) error {
	if options == nil {
		options = &PurgeOptions{}
	}
	if deployments != nil && len(deployments) == 0 {
		deployments = nil
	}

	// Execute before deployments purge hook
	err := hook.ExecuteHooks(ctx, nil, "before:purge")
	if err != nil {
		return err
	}

	// Check if root name is defined
	rootName, ok := values.RootNameFrom(ctx.Context())
	if !ok {
		options.ForcePurge = true
	}

	// Reverse them
	deploymentCaches := ctx.Config().RemoteCache().ListDeployments()
	for i := len(deploymentCaches) - 1; i >= 0; i-- {
		// Deployment cache
		deploymentCache := deploymentCaches[i]

		// Check if we should skip deleting deployment
		if deployments != nil {
			found := false
			for _, value := range deployments {
				if value == deploymentCache.Name {
					found = true
					break
				}
			}

			if !found {
				continue
			}
		}
		ctx := ctx.WithLogger(ctx.Log().WithPrefix("purge:" + deploymentCache.Name + " "))

		// Execute before deployment purge hook
		err = hook.ExecuteHooks(ctx, map[string]interface{}{
			"DEPLOY_NAME":   deploymentCache.Name,
			"DEPLOY_CONFIG": deploymentCache,
		}, hook.EventsForSingle("before:purge", deploymentCache.Name).With("deploy.beforePurge")...)
		if err != nil {
			return err
		}

		// Check if we should skip deletion
		if !options.ForcePurge && len(deploymentCache.Projects) > 0 && (len(deploymentCache.Projects) > 1 || deploymentCache.Projects[0] != rootName) {
			newProjects := []string{}
			for _, p := range deploymentCache.Projects {
				if p == rootName {
					continue
				}

				newProjects = append(newProjects, p)
			}

			deploymentCache.Projects = newProjects
			ctx.Log().Infof("Skip purging deployment %s as it is still in use by other DevSpace project(s) '%s'. Run with '--force-purge' to force deletion", deploymentCache.Name, strings.Join(deploymentCache.Projects, "', '"))
			ctx.Config().RemoteCache().SetDeployment(deploymentCache.Name, deploymentCache)
			continue
		}

		// Delete kubectl engine
		ctx.Log().Info("Deleting deployment " + deploymentCache.Name + "...")
		if deploymentCache.Kubectl != nil {
			// Purge Kubectl Deployment
			err = kubectl.Delete(ctx, deploymentCache.Name)
		} else if deploymentCache.Helm != nil {
			// Purge Helm Deployment
			err = helm.Delete(ctx, deploymentCache.Name)
		} else if deploymentCache.Tanka != nil {
			// Purge Tanka Deployment
			err = tanka.Purge(ctx, deploymentCache.Name)
		} else {
			ctx.Log().Errorf("error purging: deployment %s has no deployment method", deploymentCache.Name)
			ctx.Config().RemoteCache().DeleteDeployment(deploymentCache.Name)
			continue
		}
		if err != nil {
			// Execute on error deployment purge hook
			hookErr := hook.ExecuteHooks(ctx, map[string]interface{}{
				"DEPLOY_NAME":   deploymentCache.Name,
				"DEPLOY_CONFIG": deploymentCache,
				"ERROR":         err,
			}, hook.EventsForSingle("error:purge", deploymentCache.Name).With("deploy.errorPurge")...)
			if hookErr != nil {
				return hookErr
			}

			ctx.Log().Warnf("Error deleting deployment %s: %v", deploymentCache.Name, err)
		} else {
			err = hook.ExecuteHooks(ctx, map[string]interface{}{
				"DEPLOY_NAME":   deploymentCache.Name,
				"DEPLOY_CONFIG": deploymentCache,
			}, hook.EventsForSingle("after:purge", deploymentCache.Name).With("deploy.afterPurge")...)
			if err != nil {
				return err
			}

			ctx.Log().Donef("Successfully deleted deployment %s", deploymentCache.Name)
		}

		ctx.Config().RemoteCache().DeleteDeployment(deploymentCache.Name)
	}

	err = ctx.Config().RemoteCache().Save(ctx.Context(), ctx.KubeClient())
	if err != nil {
		return err
	}

	// Execute after deployments purge hook
	err = hook.ExecuteHooks(ctx, nil, "after:purge")
	if err != nil {
		return err
	}

	return nil
}
