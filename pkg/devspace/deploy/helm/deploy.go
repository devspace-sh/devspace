package helm

import (
	"fmt"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/helm/merge"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	hashpkg "github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
)

// Deploy deploys the given deployment with helm
func (d *DeployConfig) Deploy(cache *generated.CacheConfig, forceDeploy bool, builtImages map[string]string) (bool, error) {
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
	deployCache := cache.GetDeploymentCache(d.DeploymentConfig.Name)

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
		d.Helm, err = helm.NewClient(d.config, d.DeploymentConfig, d.Kube, d.TillerNamespace, false, d.Log)
		if err != nil {
			return false, errors.Errorf("Error creating helm client: %v", err)
		}
	}

	// Check if redeploying is necessary
	forceDeploy = forceDeploy || deployCache.HelmOverridesHash != helmOverridesHash || deployCache.HelmChartHash != hash || deployCache.DeploymentConfigHash != deploymentConfigHash
	if forceDeploy == false {
		releases, err := d.Helm.ListReleases()
		if err != nil {
			return false, err
		}

		forceDeploy = true
		for _, release := range releases {
			if release.Name == releaseName {
				forceDeploy = false
				break
			}
		}
	}

	// Deploy
	wasDeployed, err := d.internalDeploy(cache, forceDeploy, builtImages)
	if err != nil {
		return false, err
	}

	if wasDeployed {
		// Update config
		deployCache.HelmChartHash = hash
		deployCache.DeploymentConfigHash = deploymentConfigHash
		deployCache.HelmOverridesHash = helmOverridesHash
	} else {
		return false, nil
	}

	return true, nil
}

func (d *DeployConfig) internalDeploy(cache *generated.CacheConfig, forceDeploy bool, builtImages map[string]string) (bool, error) {
	var (
		releaseName     = d.DeploymentConfig.Name
		chartPath       = d.DeploymentConfig.Helm.Chart.Name
		chartValuesPath = filepath.Join(chartPath, "values.yaml")
		overwriteValues = map[interface{}]interface{}{}
	)

	// Get release namespace
	releaseNamespace := d.DeploymentConfig.Namespace

	// Check if its a local chart
	_, err := os.Stat(chartValuesPath)
	if err == nil {
		// Get values yaml when chart is locally
		_, err := os.Stat(chartValuesPath)
		if err == nil {
			err := yamlutil.ReadYamlFromFile(chartValuesPath, overwriteValues)
			if err != nil {
				return false, errors.Errorf("Couldn't deploy chart, error reading from chart values %s: %v", chartValuesPath, err)
			}
		}
	}

	// Load override values from path
	if d.DeploymentConfig.Helm.ValuesFiles != nil {
		for _, overridePath := range d.DeploymentConfig.Helm.ValuesFiles {
			overwriteValuesPath, err := filepath.Abs(overridePath)
			if err != nil {
				return false, errors.Errorf("Error retrieving absolute path from %s: %v", overridePath, err)
			}

			overwriteValuesFromPath := map[interface{}]interface{}{}
			err = yamlutil.ReadYamlFromFile(overwriteValuesPath, overwriteValuesFromPath)
			if err != nil {
				d.Log.Warnf("Error reading from chart dev overwrite values %s: %v", overwriteValuesPath, err)
			}

			merge.Values(overwriteValues).MergeInto(overwriteValuesFromPath)
		}
	}

	// Load override values from data and merge them
	if d.DeploymentConfig.Helm.Values != nil {
		merge.Values(overwriteValues).MergeInto(d.DeploymentConfig.Helm.Values)
	}

	// Add devspace specific values
	if d.DeploymentConfig.Helm.ReplaceImageTags == nil || *d.DeploymentConfig.Helm.ReplaceImageTags == true {
		// Replace image names
		shouldRedeploy := replaceContainerNames(overwriteValues, cache, d.config.Images, builtImages)
		if forceDeploy == false && shouldRedeploy {
			forceDeploy = true
		}
	}

	// Deployment is not necessary
	if forceDeploy == false {
		return false, nil
	}

	d.Log.StartWait(fmt.Sprintf("Deploying chart %s (%s) with helm", d.DeploymentConfig.Helm.Chart.Name, d.DeploymentConfig.Name))
	defer d.Log.StopWait()

	// Deploy chart
	appRelease, err := d.Helm.InstallChart(releaseName, releaseNamespace, overwriteValues, d.DeploymentConfig.Helm)
	if err != nil {
		return false, errors.Errorf("Unable to deploy helm chart: %v\nRun `%s` and `%s` to recreate the chart", err, ansi.Color("devspace purge -d "+d.DeploymentConfig.Name, "white+b"), ansi.Color("devspace deploy", "white+b"))
	}

	// Print revision
	if appRelease != nil {
		releaseRevision := int(appRelease.Version)
		d.Log.Donef("Deployed helm chart (Release revision: %d)", releaseRevision)
	} else {
		d.Log.Done("Deployed helm chart")
	}

	return true, nil
}

func replaceContainerNames(overwriteValues map[interface{}]interface{}, cache *generated.CacheConfig, imagesConf map[string]*latest.ImageConfig, builtImages map[string]string) bool {
	shouldRedeploy := false

	match := func(path, key, value string) bool {
		image, err := registry.GetStrippedDockerImageName(value)
		if err != nil {
			return false
		}

		// Search for image name
		for _, imageCache := range cache.Images {
			if imageCache.ImageName == image && imageCache.Tag != "" {
				if builtImages != nil {
					if _, ok := builtImages[image]; ok {
						shouldRedeploy = true
					}
				}

				return true
			}
		}

		return false
	}

	replace := func(path, value string) (interface{}, error) {
		image, err := registry.GetStrippedDockerImageName(value)
		if err != nil {
			return false, nil
		}

		// Search for image name
		for _, imageCache := range cache.Images {
			if imageCache.ImageName == image {
				return image + ":" + imageCache.Tag, nil
			}
		}

		return value, nil
	}

	// We ignore the error here because our replace function never throws an error
	_ = walk.Walk(overwriteValues, match, replace)

	return shouldRedeploy
}
