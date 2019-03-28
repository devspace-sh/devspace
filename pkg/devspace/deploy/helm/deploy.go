package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	hashpkg "github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
)

// Deploy deploys the given deployment with helm
func (d *DeployConfig) Deploy(generatedConfig *generated.Config, isDev, forceDeploy bool) error {
	var (
		releaseName = *d.DeploymentConfig.Name
		chartPath   = *d.DeploymentConfig.Helm.Chart.Name
		hash        = ""
	)

	// Retrieve active generated config
	activeConfig := generatedConfig.GetActive().Deploy
	if isDev {
		activeConfig = generatedConfig.GetActive().Dev
	}

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
	if _, ok := activeConfig.Deployments[*d.DeploymentConfig.Name]; ok == false {
		activeConfig.Deployments[*d.DeploymentConfig.Name] = &generated.DeploymentConfig{
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

			if activeConfig.Deployments[*d.DeploymentConfig.Name].HelmOverrideTimestamps[*override] != stat.ModTime().Unix() {
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

	// Get HelmClient
	helmClient, err := helm.NewClient(d.TillerNamespace, d.Log, false)
	if err != nil {
		return fmt.Errorf("Error creating helm client: %v", err)
	}

	// Check if redeploying is necessary
	reDeploy := forceDeploy || activeConfig.Deployments[*d.DeploymentConfig.Name].HelmChartHash != hash || activeConfig.Deployments[*d.DeploymentConfig.Name].DeploymentConfigHash != deploymentConfigHash || overrideChanged
	if reDeploy == false {
		releases, err := helmClient.Client.ListReleases()
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

	// Check if re-deployment is necessary
	if reDeploy {
		err = d.internalDeploy(generatedConfig, helmClient, isDev)
		if err != nil {
			return err
		}

		// Update config
		activeConfig.Deployments[*d.DeploymentConfig.Name].HelmChartHash = hash
		activeConfig.Deployments[*d.DeploymentConfig.Name].DeploymentConfigHash = deploymentConfigHash

		if d.DeploymentConfig.Helm.ValuesFiles != nil {
			for _, override := range *d.DeploymentConfig.Helm.ValuesFiles {
				stat, err := os.Stat(*override)
				if err != nil {
					return fmt.Errorf("Error stating override file: %s", *override)
				}

				activeConfig.Deployments[*d.DeploymentConfig.Name].HelmOverrideTimestamps[*override] = stat.ModTime().Unix()
			}
		}
	} else {
		d.Log.Infof("Skipping chart %s", chartPath)
	}

	return nil
}

func (d *DeployConfig) internalDeploy(generatedConfig *generated.Config, helmClient *helm.ClientWrapper, isDev bool) error {
	d.Log.StartWait("Deploying helm chart")
	defer d.Log.StopWait()

	var (
		config          = configutil.GetConfig()
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
				return fmt.Errorf("Couldn't deploy chart, error reading from chart values %s: %v", chartValuesPath, err)
			}
		}
	}

	// Load override values from path
	if d.DeploymentConfig.Helm.ValuesFiles != nil {
		for _, overridePath := range *d.DeploymentConfig.Helm.ValuesFiles {
			overwriteValuesPath, err := filepath.Abs(*overridePath)
			if err != nil {
				return fmt.Errorf("Error retrieving absolute path from %s: %v", *overridePath, err)
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
		replaceContainerNames(overwriteValues, generatedConfig, isDev)

		// Set images and pull secrets values
		overwriteValues["images"] = getImageValues(config, generatedConfig, isDev)
		overwriteValues["pullSecrets"] = getPullSecrets(overwriteValues, overwriteValues, config)
	}

	// Deploy chart
	appRelease, err := helmClient.InstallChart(releaseName, releaseNamespace, &overwriteValues, d.DeploymentConfig.Helm)
	if err != nil {
		return fmt.Errorf("Unable to deploy helm chart: %v", err)
	}

	// Print revision
	if appRelease != nil {
		releaseRevision := int(appRelease.Version)
		d.Log.Donef("Deployed helm chart (Release revision: %d)", releaseRevision)
	} else {
		d.Log.Done("Deployed helm chart")
	}

	return nil
}

func getImageValues(config *v1.Config, generatedConfig *generated.Config, isDev bool) map[interface{}]interface{} {
	active := generatedConfig.GetActive()

	var tags map[string]string
	if isDev {
		tags = active.Dev.ImageTags
	} else {
		tags = active.Deploy.ImageTags
	}

	overwriteContainerValues := map[interface{}]interface{}{}
	if config.Images != nil {
		for imageName, imageConf := range *config.Images {
			tag := tags[*imageConf.Image]
			if imageConf.Tag != nil {
				tag = *imageConf.Tag
			}

			overwriteContainerValues[imageName] = map[interface{}]interface{}{
				"image": *imageConf.Image + ":" + tag,
				"tag":   tag,
				"repo":  *imageConf.Image,
			}
		}
	}

	return overwriteContainerValues
}

func replaceContainerNames(overwriteValues map[interface{}]interface{}, generatedConfig *generated.Config, isDev bool) {
	active := generatedConfig.GetActive()

	var tags map[string]string
	if isDev {
		tags = active.Dev.ImageTags
	} else {
		tags = active.Deploy.ImageTags
	}

	match := func(key, value string) bool {
		value = strings.TrimSpace(value)

		image := strings.Split(value, ":")
		if _, ok := tags[image[0]]; ok {
			return true
		}

		return false
	}

	replace := func(value string) interface{} {
		value = strings.TrimSpace(value)

		image := strings.Split(value, ":")
		return image[0] + ":" + tags[image[0]]
	}

	walk.Walk(overwriteValues, match, replace)
}

func getPullSecrets(values, overwriteValues map[interface{}]interface{}, config *v1.Config) []interface{} {
	overwritePullSecrets := []interface{}{}
	overwritePullSecretsFromFile, overwritePullSecretsExisting := overwriteValues["pullSecrets"]
	if overwritePullSecretsExisting {
		overwritePullSecrets = overwritePullSecretsFromFile.([]interface{})
	}

	pullSecretsFromFile, pullSecretsExisting := values["pullSecrets"]

	if pullSecretsExisting {
		existingPullSecrets := pullSecretsFromFile.([]interface{})
		overwritePullSecrets = append(overwritePullSecrets, existingPullSecrets...)
	}

	for _, autoGeneratedPullSecret := range registry.GetPullSecretNames() {
		overwritePullSecrets = append(overwritePullSecrets, autoGeneratedPullSecret)
	}

	return overwritePullSecrets
}
