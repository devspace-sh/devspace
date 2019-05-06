package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm"
	hashpkg "github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
)

// Deploy deploys the given deployment with helm
func (d *DeployConfig) Deploy(cache *generated.CacheConfig, forceDeploy bool, builtImages map[string]string) error {
	var (
		releaseName = *d.DeploymentConfig.Name
		chartPath   = *d.DeploymentConfig.Helm.Chart.Name
		hash        = ""
	)

	// Hash the chart directory if there is any
	_, err := os.Stat(chartPath)
	if err == nil {
		// Check if the chart directory has changed
		hash, err = hashpkg.Directory(chartPath)
		if err != nil {
			return fmt.Errorf("Error hashing chart directory: %v", err)
		}
	}

	// Ensure deployment config is there
	if _, ok := cache.Deployments[*d.DeploymentConfig.Name]; ok == false {
		cache.Deployments[*d.DeploymentConfig.Name] = &generated.DeploymentConfig{
			HelmOverrideTimestamps: make(map[string]int64),
		}
	}

	// Check values files for changes
	overrideChanged := false
	if d.DeploymentConfig.Helm.ValuesFiles != nil {
		for _, override := range *d.DeploymentConfig.Helm.ValuesFiles {
			stat, err := os.Stat(*override)
			if err != nil {
				return fmt.Errorf("Error stating override file: %s", *override)
			}

			if cache.Deployments[*d.DeploymentConfig.Name].HelmOverrideTimestamps[*override] != stat.ModTime().Unix() {
				overrideChanged = true
				break
			}
		}
	}

	// Check deployment config for changes
	configStr, err := yaml.Marshal(d.DeploymentConfig)
	if err != nil {
		return errors.Wrap(err, "marshal deployment config")
	}

	deploymentConfigHash := hashpkg.String(string(configStr))

	// Get HelmClient if necessary
	if d.Helm == nil {
		d.Helm, err = helm.NewClient(d.TillerNamespace, d.Log, false)
		if err != nil {
			return fmt.Errorf("Error creating helm client: %v", err)
		}
	}

	// Check if redeploying is necessary
	reDeploy := forceDeploy || cache.Deployments[*d.DeploymentConfig.Name].HelmChartHash != hash || cache.Deployments[*d.DeploymentConfig.Name].DeploymentConfigHash != deploymentConfigHash || overrideChanged
	if reDeploy == false {
		releases, err := d.Helm.ListReleases()
		if err != nil {
			return err
		}

		reDeploy = true
		if releases != nil {
			for _, release := range releases.Releases {
				if release.GetName() == releaseName {
					reDeploy = false
					break
				}
			}
		}
	}

	// Deploy
	wasDeployed, err := d.internalDeploy(cache, reDeploy, builtImages)
	if err != nil {
		return err
	}

	if wasDeployed {
		// Update config
		cache.Deployments[*d.DeploymentConfig.Name].HelmChartHash = hash
		cache.Deployments[*d.DeploymentConfig.Name].DeploymentConfigHash = deploymentConfigHash

		if d.DeploymentConfig.Helm.ValuesFiles != nil {
			for _, override := range *d.DeploymentConfig.Helm.ValuesFiles {
				stat, err := os.Stat(*override)
				if err != nil {
					return fmt.Errorf("Error stating override file: %s", *override)
				}

				cache.Deployments[*d.DeploymentConfig.Name].HelmOverrideTimestamps[*override] = stat.ModTime().Unix()
			}
		}
	} else {
		d.Log.Infof("Skipping chart %s", chartPath)
	}

	return nil
}

func (d *DeployConfig) internalDeploy(cache *generated.CacheConfig, forceDeploy bool, builtImages map[string]string) (bool, error) {
	var (
		releaseName     = *d.DeploymentConfig.Name
		chartPath       = *d.DeploymentConfig.Helm.Chart.Name
		chartValuesPath = filepath.Join(chartPath, "values.yaml")
		overwriteValues = map[interface{}]interface{}{}
	)

	// Get release namespace
	releaseNamespace := ""
	if d.DeploymentConfig.Namespace != nil {
		releaseNamespace = *d.DeploymentConfig.Namespace
	}

	// Check if its a local chart
	_, err := os.Stat(chartValuesPath)
	if err == nil {
		// Get values yaml when chart is locally
		_, err := os.Stat(chartValuesPath)
		if err == nil {
			err := yamlutil.ReadYamlFromFile(chartValuesPath, overwriteValues)
			if err != nil {
				return false, fmt.Errorf("Couldn't deploy chart, error reading from chart values %s: %v", chartValuesPath, err)
			}
		}
	}

	// Load override values from path
	if d.DeploymentConfig.Helm.ValuesFiles != nil {
		for _, overridePath := range *d.DeploymentConfig.Helm.ValuesFiles {
			overwriteValuesPath, err := filepath.Abs(*overridePath)
			if err != nil {
				return false, fmt.Errorf("Error retrieving absolute path from %s: %v", *overridePath, err)
			}

			overwriteValuesFromPath := map[interface{}]interface{}{}
			err = yamlutil.ReadYamlFromFile(overwriteValuesPath, overwriteValuesFromPath)
			if err != nil {
				d.Log.Warnf("Error reading from chart dev overwrite values %s: %v", overwriteValuesPath, err)
			}

			Values(overwriteValues).MergeInto(overwriteValuesFromPath)
		}
	}

	// Load override values from data and merge them
	if d.DeploymentConfig.Helm.Values != nil {
		Values(overwriteValues).MergeInto(*d.DeploymentConfig.Helm.Values)
	}

	// Add devspace specific values
	if d.DeploymentConfig.Helm.DevSpaceValues == nil || *d.DeploymentConfig.Helm.DevSpaceValues == true {
		// Replace image names
		shouldRedeploy := replaceContainerNames(overwriteValues, cache, builtImages)
		if forceDeploy == false && shouldRedeploy {
			forceDeploy = true
		}
	}

	// Deployment is not necessary
	if forceDeploy == false {
		return false, nil
	}

	d.Log.StartWait("Deploying helm chart")
	defer d.Log.StopWait()

	// Deploy chart
	appRelease, err := d.Helm.InstallChart(releaseName, releaseNamespace, &overwriteValues, d.DeploymentConfig.Helm)
	if err != nil {
		return false, fmt.Errorf("Unable to deploy helm chart: %v\nRun `%s` and `%s` to recreate the chart", err, ansi.Color("devspace purge -d "+*d.DeploymentConfig.Name, "white+b"), ansi.Color("devspace deploy", "white+b"))
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

func replaceContainerNames(overwriteValues map[interface{}]interface{}, cache *generated.CacheConfig, builtImages map[string]string) bool {
	tags := cache.ImageTags
	shouldRedeploy := false

	match := func(path, key, value string) bool {
		value = strings.TrimSpace(value)

		image := strings.Split(value, ":")
		if _, ok := tags[image[0]]; ok {
			if builtImages != nil {
				if _, ok := builtImages[image[0]]; ok {
					shouldRedeploy = true
				}
			}

			return true
		}

		return false
	}

	replace := func(path, value string) interface{} {
		value = strings.TrimSpace(value)

		image := strings.Split(value, ":")
		return image[0] + ":" + tags[image[0]]
	}

	walk.Walk(overwriteValues, match, replace)

	return shouldRedeploy
}
