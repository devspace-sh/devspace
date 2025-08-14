package helm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/legacy"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/util/stringutil"

	"github.com/loft-sh/devspace/pkg/devspace/helm/types"

	yaml "gopkg.in/yaml.v3"

	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm/merge"
	"github.com/loft-sh/devspace/pkg/devspace/helm"
	hashpkg "github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
)

// Deploy deploys the given deployment with helm
func (d *DeployConfig) Deploy(ctx devspacecontext.Context, forceDeploy bool) (bool, error) {
	var releaseName string
	if d.DeploymentConfig.Helm.ReleaseName != "" {
		releaseName = d.DeploymentConfig.Helm.ReleaseName
	} else {
		releaseName = d.DeploymentConfig.Name
	}

	var (
		chartPath = d.DeploymentConfig.Helm.Chart.Name
		hash      = ""
	)

	releaseNamespace := ctx.KubeClient().Namespace()
	if d.DeploymentConfig.Namespace != "" {
		releaseNamespace = d.DeploymentConfig.Namespace
	}

	if d.DeploymentConfig.Helm.Chart.Source != nil {
		downloadPath, err := d.Helm.DownloadChart(ctx, d.DeploymentConfig.Helm)
		if err != nil {
			return false, errors.Wrap(err, "download chart")
		}
		chartPath = downloadPath
	}

	// Hash the chart directory if there is any
	_, err := os.Stat(ctx.ResolvePath(chartPath))
	if err == nil {
		chartPath = ctx.ResolvePath(chartPath)

		// Check if the chart directory has changed
		hash, err = hashpkg.DirectoryExcludes(chartPath, []string{
			".git/",
			".devspace/",
		}, true)
		if err != nil {
			return false, errors.Errorf("Error hashing chart directory: %v", err)
		}
	}

	// Ensure deployment config is there
	deployCache, _ := ctx.Config().RemoteCache().GetDeployment(releaseName)

	// Check values files for changes
	helmOverridesHash := ""
	if d.DeploymentConfig.Helm.ValuesFiles != nil {
		for _, override := range d.DeploymentConfig.Helm.ValuesFiles {
			override = ctx.ResolvePath(override)

			hash, err := hashpkg.Directory(override)
			if err != nil {
				return false, errors.Errorf("Error stating override file %s: %v", override, err)
			}

			helmOverridesHash += hash
		}
	}

	// Check deployment config for changes
	configStr, err := yaml.Marshal(d.DeploymentConfig)
	if err != nil {
		return false, errors.Wrap(err, "marshal deployment config")
	}

	deploymentConfigHash := hashpkg.String(string(configStr))

	// Get HelmClient if necessary
	if d.Helm == nil {
		d.Helm, err = helm.NewClient(ctx.Log())
		if err != nil {
			return false, errors.Errorf("Error creating helm client: %v", err)
		}
	}

	// Get deployment values
	redeploy, deployValues, err := d.getDeploymentValues(ctx)
	if err != nil {
		return false, err
	}

	// Check deployment values for changes
	deployValuesBytes, err := yaml.Marshal(deployValues)
	if err != nil {
		return false, errors.Wrap(err, "marshal deployment values")
	}
	deployValuesHash := hashpkg.String(string(deployValuesBytes))

	// Check if redeploying is necessary
	helmCache := deployCache.Helm
	if helmCache == nil {
		helmCache = &remotecache.HelmCache{}
	}

	forceDeploy = forceDeploy || redeploy || deployCache.DeploymentConfigHash != deploymentConfigHash || helmCache.ValuesHash != deployValuesHash || helmCache.OverridesHash != helmOverridesHash || helmCache.ChartHash != hash
	if !forceDeploy {
		releases, err := d.Helm.ListReleases(ctx, releaseNamespace)
		if err != nil {
			return false, err
		}

		forceDeploy = true
		for _, release := range releases {
			if release.Name == releaseName && release.Revision == helmCache.ReleaseRevision {
				forceDeploy = false
				break
			}
		}
	}

	// Deploy
	if forceDeploy {
		release, err := d.internalDeploy(ctx, deployValues, nil)
		if err != nil {
			return false, err
		}

		deployCache.DeploymentConfigHash = deploymentConfigHash
		helmCache.Release = releaseName
		helmCache.ReleaseNamespace = releaseNamespace
		helmCache.ChartHash = hash
		helmCache.ValuesHash = deployValuesHash
		helmCache.OverridesHash = helmOverridesHash
		if release != nil {
			helmCache.ReleaseRevision = release.Revision
		}

		deployCache.Helm = helmCache
		if rootName, ok := values.RootNameFrom(ctx.Context()); ok && !stringutil.Contains(deployCache.Projects, rootName) {
			deployCache.Projects = append(deployCache.Projects, rootName)
		}
		ctx.Config().RemoteCache().SetDeployment(releaseName, deployCache)
		return true, nil
	}

	if rootName, ok := values.RootNameFrom(ctx.Context()); ok && !stringutil.Contains(deployCache.Projects, rootName) {
		deployCache.Projects = append(deployCache.Projects, rootName)
	}
	ctx.Config().RemoteCache().SetDeployment(releaseName, deployCache)
	return false, nil
}

func (d *DeployConfig) internalDeploy(ctx devspacecontext.Context, overwriteValues map[string]interface{}, out io.Writer) (*types.Release, error) {
	var releaseName string
	if d.DeploymentConfig.Helm.ReleaseName != "" {
		releaseName = d.DeploymentConfig.Helm.ReleaseName
	} else {
		releaseName = d.DeploymentConfig.Name
	}
	releaseNamespace := ctx.KubeClient().Namespace()
	if d.DeploymentConfig.Namespace != "" {
		releaseNamespace = d.DeploymentConfig.Namespace
	}

	if out != nil {
		str, err := d.Helm.Template(ctx, releaseName, releaseNamespace, overwriteValues, d.DeploymentConfig.Helm)
		if err != nil {
			return nil, err
		}

		_, _ = out.Write([]byte("\n" + str + "\n"))
		return nil, nil
	}

	ctx.Log().Infof("Deploying chart %s (%s) with helm...", d.DeploymentConfig.Helm.Chart.Name, releaseName)
	valuesOut, _ := yaml.Marshal(overwriteValues)
	ctx.Log().Debugf("Deploying chart with values:\n %v\n", string(valuesOut))

	// Deploy chart
	appRelease, err := d.Helm.InstallChart(ctx, releaseName, releaseNamespace, overwriteValues, d.DeploymentConfig.Helm)
	if err != nil {
		return nil, errors.Errorf("unable to deploy helm chart: %v", err)
	}

	// Print revision
	if appRelease != nil {
		ctx.Log().Donef("Deployed helm chart (Release revision: %s)", appRelease.Revision)
	} else {
		ctx.Log().Done("Deployed helm chart")
	}

	return appRelease, nil
}

func (d *DeployConfig) getDeploymentValues(ctx devspacecontext.Context) (bool, map[string]interface{}, error) {
	var (
		chartPath       = d.DeploymentConfig.Helm.Chart.Name
		chartValuesPath = ctx.ResolvePath(filepath.Join(chartPath, "values.yaml"))
		overwriteValues = map[string]interface{}{}
		shouldRedeploy  = false
	)

	// Check if its a local chart
	_, err := os.Stat(chartValuesPath)
	if err == nil {
		err := yamlutil.ReadYamlFromFile(chartValuesPath, overwriteValues)
		if err != nil {
			return false, nil, errors.Errorf("Couldn't deploy chart, error reading from chart values %s: %v", chartValuesPath, err)
		}

		if d.DeploymentConfig.UpdateImageTags == nil || *d.DeploymentConfig.UpdateImageTags {
			redeploy, err := legacy.ReplaceImageNames(overwriteValues, ctx.Config(), ctx.Dependencies(), nil)
			if err != nil {
				return false, nil, err
			}
			shouldRedeploy = shouldRedeploy || redeploy
		}
	}

	// Load override values from path
	if d.DeploymentConfig.Helm.ValuesFiles != nil {
		for _, overridePath := range d.DeploymentConfig.Helm.ValuesFiles {
			overwriteValuesPath := ctx.ResolvePath(overridePath)
			overwriteValuesFromPath := map[string]interface{}{}
			err = yamlutil.ReadYamlFromFile(overwriteValuesPath, overwriteValuesFromPath)
			if err != nil {
				return false, nil, fmt.Errorf("error reading from chart dev overwrite values %s: %v", overwriteValuesPath, err)
			}

			// Replace image names
			if d.DeploymentConfig.UpdateImageTags == nil || *d.DeploymentConfig.UpdateImageTags {
				redeploy, err := legacy.ReplaceImageNames(overwriteValuesFromPath, ctx.Config(), ctx.Dependencies(), nil)
				if err != nil {
					return false, nil, err
				}
				shouldRedeploy = shouldRedeploy || redeploy
			}

			merge.Values(overwriteValues).MergeInto(overwriteValuesFromPath)
		}
	}

	// Load override values from data and merge them
	if d.DeploymentConfig.Helm.Values != nil {
		enableLegacy := false
		if d.DeploymentConfig.UpdateImageTags == nil || *d.DeploymentConfig.UpdateImageTags {
			enableLegacy = true
		}
		redeploy, _, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir(), enableLegacy).FillRuntimeVariablesWithRebuild(ctx.Context(), d.DeploymentConfig.Helm.Values, ctx.Config(), ctx.Dependencies())
		if err != nil {
			return false, nil, err
		}
		shouldRedeploy = shouldRedeploy || redeploy

		merge.Values(overwriteValues).MergeInto(d.DeploymentConfig.Helm.Values)
	}

	// Validate deployment values
	err = versions.ValidateComponentConfig(d.DeploymentConfig, overwriteValues)
	if err != nil {
		return false, nil, err
	}

	return shouldRedeploy, overwriteValues, nil
}
