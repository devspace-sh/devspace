package dependency

import (
	"github.com/loft-sh/devspace/pkg/devspace/build"
	buildtypes "github.com/loft-sh/devspace/pkg/devspace/build/types"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"io"
	"os"

	"github.com/pkg/errors"
)

// Dependency holds the dependency config and has an id
type Dependency struct {
	name         string
	absolutePath string
	root         bool
	localConfig  config.Config

	children []types.Dependency

	dependencyConfig *latest.DependencyConfig
	dependencyCache  localcache.Cache

	kubeClient     kubectl.Client
	dockerClient   docker.Client
	registryClient pullsecrets.Client
}

// Implement Interface Methods

func (d *Dependency) Name() string { return d.name }

func (d *Dependency) Root() bool { return d.root }

func (d *Dependency) KubeClient() kubectl.Client { return d.kubeClient }

func (d *Dependency) Config() config.Config { return d.localConfig }

func (d *Dependency) Path() string { return d.absolutePath }

func (d *Dependency) DependencyConfig() *latest.DependencyConfig { return d.dependencyConfig }

func (d *Dependency) Children() []types.Dependency { return d.children }

func (d *Dependency) Command(command string, args []string) error {
	return ExecuteCommand(d.localConfig.Config().Commands, command, args, d.absolutePath, os.Stdout, os.Stderr)
}

// Build builds and pushes all defined images
func (d *Dependency) Build(ctx *devspacecontext.Context, buildOptions *build.Options) error {
	return d.buildImages(ctx, false, buildOptions)
}

// Deploy deploys the dependency if necessary
func (d *Dependency) Deploy(ctx *devspacecontext.Context, skipBuild, skipDeploy, forceDeploy bool, buildOptions *build.Options) error {
	// Create namespace if necessary
	err := d.kubeClient.EnsureNamespace(ctx.Context, ctx.KubeClient.Namespace(), ctx.Log)
	if err != nil {
		return errors.Errorf("unable to create namespace: %v", err)
	}

	// Create pull secrets and private registry if necessary
	err = d.registryClient.EnsurePullSecrets(ctx, ctx.KubeClient.Namespace())
	if err != nil {
		ctx.Log.Warn(err)
	}

	// TODO: start pipeline here

	// Check if image build is enabled
	err = d.buildImages(ctx, skipBuild, buildOptions)
	if err != nil {
		return err
	}

	// Deploy all defined deployments
	if !skipDeploy {
		err = deploy.NewController().Deploy(ctx, nil, &deploy.Options{
			ForceDeploy: forceDeploy,
		})
		if err != nil {
			return err
		}

		// Save Config
		err = ctx.Config.RemoteCache().Save(ctx.Context, ctx.KubeClient)
		if err != nil {
			return errors.Errorf("Error saving generated config: %v", err)
		}
	}

	return nil
}

// Render renders the dependency
func (d *Dependency) Render(ctx *devspacecontext.Context, skipBuild bool, buildOptions *build.Options, out io.Writer) error {
	// Check if image build is enabled
	err := d.buildImages(ctx, skipBuild, buildOptions)
	if err != nil {
		return err
	}

	// Deploy all defined deployments
	return deploy.NewController().Render(ctx, nil, &deploy.Options{}, out)
}

// Purge purges the dependency
func (d *Dependency) Purge(ctx *devspacecontext.Context) error {
	// Purge the deployments
	err := deploy.NewController().Purge(ctx, nil)
	if err != nil {
		ctx.Log.Errorf("error purging dependency %s: %v", d.Name(), err)
	}

	err = ctx.Config.RemoteCache().Save(ctx.Context, ctx.KubeClient)
	if err != nil {
		ctx.Log.Errorf("error saving remote cache: %v", err)
	}

	return nil
}

func (d *Dependency) buildImages(ctx *devspacecontext.Context, skipBuild bool, buildOptions *build.Options) error {
	// Check if image build is enabled
	if !skipBuild && !d.dependencyConfig.SkipBuild {
		// Build images
		err := build.NewController().Build(ctx, nil, buildOptions)
		if err != nil {
			return err
		}

		// merge built images
		builtImages, ok := ctx.Config.GetRuntimeVariable(constants.BuiltImagesKey)
		if ok {
			builtImagesMap, ok := builtImages.(map[string]buildtypes.ImageNameTag)
			if ok && len(builtImagesMap) > 0 && ctx.Config != nil && ctx.Config.LocalCache() != nil {
				err = ctx.Config.LocalCache().Save()
				if err != nil {
					return errors.Errorf("error saving local cache: %v", err)
				}
			}
		}
	}

	return nil
}

func skipDependency(name string, skipDependencies []string) bool {
	for _, sd := range skipDependencies {
		if sd == name {
			return true
		}
	}
	return false
}

func foundDependency(name string, dependencies []string) bool {
	if len(dependencies) == 0 {
		return true
	}

	for _, n := range dependencies {
		if n == name {
			return true
		}
	}

	return false
}
