package tanka

import (
	"io"
	"strings"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"

	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
)

const (
	// variables available to the tanka environment when using DevSpace
	ExtVarNamespace = "DEVSPACE_NAMESPACE"
	ExtVarAPIServer = "DEVSPACE_API_SERVER"
	ExtVarName      = "DEVSPACE_NAME"
)

// DeployConfig holds the necessary information for tanka deployment
type DeployConfig struct {
	// TODO JsonnetBundlerBinaryPath string
	name   string
	target string

	tankaEnv TankaEnvironment

	// used for internal caching on purge only! Do not use directly as schema can change
	tankaConfig *latest.TankaConfig
}

// New creates a new deploy config for tanka
func New(ctx devspacecontext.Context, deployConfig *latest.DeploymentConfig) (deployer.Interface, error) {
	hydrate := map[string]string{}

	if err := validateConfig(deployConfig); err != nil {
		return nil, err
	}

	if deployConfig.Tanka.ExternalStringVariables == nil {
		deployConfig.Tanka.ExternalStringVariables = make(map[string]string)
	}

	// hydrate from deployConfig
	hydrate[ExtVarName] = deployConfig.Name
	hydrate[ExtVarNamespace] = deployConfig.Namespace

	// hydrate tanka variables from Kubeconfig
	if client := ctx.KubeClient(); client != nil {

		// hydrate namespace if not set
		if hydrate[ExtVarNamespace] == "" {
			hydrate[ExtVarNamespace] = client.Namespace()
		}

		// hydrate APIServer
		hydrate[ExtVarAPIServer] = client.RestConfig().Host
	}

	// merge hydrated
	for k, v := range hydrate {
		deployConfig.Tanka.ExternalStringVariables[k] = v
	}

	cfg := &DeployConfig{
		name:        deployConfig.Name,
		target:      deployConfig.Tanka.Target,
		tankaEnv:    NewTankaEnvironment(deployConfig.Tanka),
		tankaConfig: deployConfig.Tanka,
	}

	return cfg, nil
}

// Render writes runs `tk show` and outputs it to the CLI.
func (d *DeployConfig) Render(ctx devspacecontext.Context, out io.Writer) error {
	return d.tankaEnv.Show(ctx, out)
}

// Status runs `tk diff` to view changes between manifests and the deployed state.
func (d *DeployConfig) Status(ctx devspacecontext.Context) (*deployer.StatusResult, error) {
	diff, err := d.tankaEnv.Diff(ctx)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(diff, "No differences.") {
		return &deployer.StatusResult{
			Name:   d.name,
			Type:   "Tanka",
			Target: d.target,
			Status: "no diff",
		}, nil
	}
	return &deployer.StatusResult{
		Name:   d.name,
		Type:   "Tanka",
		Target: d.target,
		Status: diff,
	}, nil

}

// Deploy runs `tk apply` to apply local manifests to the cluster.
func (d *DeployConfig) Deploy(ctx devspacecontext.Context, _ bool) (bool, error) {
	var err error

	deployCache, _ := ctx.Config().RemoteCache().GetDeployment(d.name)

	// as devspace does not pass the original context on the purge option,
	// we'll store it in the remote cache
	deployCache.Name = d.name
	deployCache.Tanka = &remotecache.TankaCache{
		AppliedTankaConfig: d.tankaConfig,
	}
	ctx.Config().RemoteCache().SetDeployment(d.name, deployCache)

	// Check if we need to run jb install
	if d.tankaConfig.RunJsonnetBundlerInstall == nil || *d.tankaConfig.RunJsonnetBundlerInstall {
		err = d.tankaEnv.Install(ctx)
		if err != nil {
			return false, err
		}

	}

	if d.tankaConfig.RunJsonnetBundlerUpdate == nil || *d.tankaConfig.RunJsonnetBundlerUpdate {
		err = d.tankaEnv.Update(ctx)
		if err != nil {
			return false, err
		}
	}
	// Delete orphaned resources
	if err := d.tankaEnv.Prune(ctx); err != nil {
		return false, err
	}
	// Apply the desired resources
	if err := d.tankaEnv.Apply(ctx); err != nil {
		return false, err
	}
	return true, nil
}
