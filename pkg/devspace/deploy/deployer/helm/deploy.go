package helm

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/legacy"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"io"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/pkg/devspace/helm/types"

	yaml "gopkg.in/yaml.v2"

	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm/merge"
	"github.com/loft-sh/devspace/pkg/devspace/helm"
	hashpkg "github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
)

// Deploy deploys the given deployment with helm
func (d *DeployConfig) Deploy(forceDeploy bool, builtImages map[string]string) (bool, error) {
	var (
		releaseName = d.DeploymentConfig.Name
		chartPath   = d.DeploymentConfig.Helm.Chart.Name
		hash        = ""
	)

	// Hash the chart directory if there is any
	_, err := os.Stat(chartPath)
	if err == nil {
		// Check if the chart directory has changed
		hash, err = hashpkg.Directory(chartPath)
		if err != nil {
			return false, errors.Errorf("Error hashing chart directory: %v", err)
		}
	}

	// Ensure deployment config is there
	deployCache := d.config.Generated().GetActive().GetDeploymentCache(d.DeploymentConfig.Name)

	// Check values files for changes
	helmOverridesHash := ""
	if d.DeploymentConfig.Helm.ValuesFiles != nil {
		for _, override := range d.DeploymentConfig.Helm.ValuesFiles {
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
		d.Helm, err = helm.NewClient(d.config.Config(), d.DeploymentConfig, d.Kube, d.TillerNamespace, false, false, d.Log)
		if err != nil {
			return false, errors.Errorf("Error creating helm client: %v", err)
		}
	}

	// Get deployment values
	redeploy, deployValues, err := d.getDeploymentValues(builtImages)
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
	forceDeploy = forceDeploy || redeploy || deployCache.HelmValuesHash != deployValuesHash || deployCache.HelmOverridesHash != helmOverridesHash || deployCache.HelmChartHash != hash || deployCache.DeploymentConfigHash != deploymentConfigHash
	if !forceDeploy {
		releases, err := d.Helm.ListReleases(d.DeploymentConfig.Helm)
		if err != nil {
			return false, err
		}

		forceDeploy = true
		for _, release := range releases {
			if release.Name == releaseName && release.Revision == deployCache.HelmReleaseRevision {
				forceDeploy = false
				break
			}
		}
	}

	// Deploy
	if forceDeploy {
		release, err := d.internalDeploy(deployValues, nil)
		if err != nil {
			return false, err
		}

		deployCache.DeploymentConfigHash = deploymentConfigHash
		deployCache.HelmChartHash = hash
		deployCache.HelmValuesHash = deployValuesHash
		deployCache.HelmOverridesHash = helmOverridesHash
		if release != nil {
			deployCache.HelmReleaseRevision = release.Revision
		}

		return true, nil
	}

	return false, nil
}

func (d *DeployConfig) internalDeploy(overwriteValues map[interface{}]interface{}, out io.Writer) (*types.Release, error) {
	var (
		releaseName      = d.DeploymentConfig.Name
		releaseNamespace = d.DeploymentConfig.Namespace
	)

	if out != nil {
		str, err := d.Helm.Template(releaseName, releaseNamespace, overwriteValues, d.DeploymentConfig.Helm)
		if err != nil {
			return nil, err
		}

		_, _ = out.Write([]byte("\n" + str + "\n"))
		return nil, nil
	}

	d.Log.StartWait(fmt.Sprintf("Deploying chart %s (%s) with helm", d.DeploymentConfig.Helm.Chart.Name, d.DeploymentConfig.Name))
	defer d.Log.StopWait()

	// Deploy chart
	appRelease, err := d.Helm.InstallChart(releaseName, releaseNamespace, overwriteValues, d.DeploymentConfig.Helm)
	if err != nil {
		return nil, errors.Errorf("Unable to deploy helm chart: %v\nRun `%s` and `%s` to recreate the chart", err, ansi.Color("devspace purge -d "+d.DeploymentConfig.Name, "white+b"), ansi.Color("devspace deploy", "white+b"))
	}

	// Print revision
	if appRelease != nil {
		d.Log.Donef("Deployed helm chart (Release revision: %s)", appRelease.Revision)
	} else {
		d.Log.Done("Deployed helm chart")
	}

	return appRelease, nil
}

func (d *DeployConfig) getDeploymentValues(builtImages map[string]string) (bool, map[interface{}]interface{}, error) {
	var (
		chartPath       = d.DeploymentConfig.Helm.Chart.Name
		chartValuesPath = filepath.Join(chartPath, "values.yaml")
		overwriteValues = map[interface{}]interface{}{}
		shouldRedeploy  = false
	)

	// Check if its a local chart
	_, err := os.Stat(chartValuesPath)
	if err == nil {
		err := yamlutil.ReadYamlFromFile(chartValuesPath, overwriteValues)
		if err != nil {
			return false, nil, errors.Errorf("Couldn't deploy chart, error reading from chart values %s: %v", chartValuesPath, err)
		}

		if d.DeploymentConfig.Helm.ReplaceImageTags == nil || *d.DeploymentConfig.Helm.ReplaceImageTags {
			redeploy, err := legacy.ReplaceImageNames(overwriteValues, d.config, d.dependencies, builtImages, nil)
			if err != nil {
				return false, nil, err
			}
			shouldRedeploy = shouldRedeploy || redeploy
		}
	}

	// Load override values from path
	if d.DeploymentConfig.Helm.ValuesFiles != nil {
		for _, overridePath := range d.DeploymentConfig.Helm.ValuesFiles {
			overwriteValuesPath, err := filepath.Abs(overridePath)
			if err != nil {
				return false, nil, errors.Errorf("Error retrieving absolute path from %s: %v", overridePath, err)
			}

			overwriteValuesFromPath := map[interface{}]interface{}{}
			err = yamlutil.ReadYamlFromFile(overwriteValuesPath, overwriteValuesFromPath)
			if err != nil {
				d.Log.Warnf("Error reading from chart dev overwrite values %s: %v", overwriteValuesPath, err)
			}

			// Replace image names
			if d.DeploymentConfig.Helm.ReplaceImageTags == nil || *d.DeploymentConfig.Helm.ReplaceImageTags {
				redeploy, err := legacy.ReplaceImageNames(overwriteValuesFromPath, d.config, d.dependencies, builtImages, nil)
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
		if d.DeploymentConfig.Helm.ReplaceImageTags == nil || *d.DeploymentConfig.Helm.ReplaceImageTags {
			enableLegacy = true
		}
		redeploy, _, err := runtimevar.NewRuntimeResolver(enableLegacy).FillRuntimeVariablesWithRebuild(d.DeploymentConfig.Helm.Values, d.config, d.dependencies, builtImages)
		if err != nil {
			return false, nil, err
		}
		shouldRedeploy = shouldRedeploy || redeploy

		merge.Values(overwriteValues).MergeInto(d.DeploymentConfig.Helm.Values)
	}

	return shouldRedeploy, overwriteValues, nil
}
