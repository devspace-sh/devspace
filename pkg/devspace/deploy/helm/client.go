package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/devspace/registry"
	"github.com/covexo/devspace/pkg/util/hash"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/yamlutil"
	"k8s.io/client-go/kubernetes"
)

// DeployConfig holds the information necessary to deploy via helm
type DeployConfig struct {
	KubeClient       *kubernetes.Clientset
	TillerNamespace  string
	DeploymentConfig *v1.DeploymentConfig
	UseDevOverwrite  bool
	Log              log.Logger
}

// New creates a new helm deployment client
func New(kubectl *kubernetes.Clientset, deployConfig *v1.DeploymentConfig, useDevOverwrite bool, log log.Logger) (*DeployConfig, error) {
	config := configutil.GetConfig()
	return &DeployConfig{
		KubeClient:       kubectl,
		TillerNamespace:  *config.Tiller.Namespace,
		DeploymentConfig: deployConfig,
		UseDevOverwrite:  useDevOverwrite,
		Log:              log,
	}, nil
}

// Delete deletes the release
func (d *DeployConfig) Delete() error {
	// Delete with helm engine
	isDeployed := helm.IsTillerDeployed(d.KubeClient)
	if isDeployed == false {
		return nil
	}

	// Get HelmClient
	helmClient, err := helm.NewClient(d.KubeClient, d.Log, false)
	if err != nil {
		return err
	}

	_, err = helmClient.DeleteRelease(*d.DeploymentConfig.Name, true)
	if err != nil {
		return err
	}

	return nil
}

// Status gets the status of the deployment
func (d *DeployConfig) Status() ([][]string, error) {
	var values [][]string

	// Get HelmClient
	helmClient, err := helm.NewClient(d.KubeClient, d.Log, false)
	if err != nil {
		return nil, err
	}

	releases, err := helmClient.Client.ListReleases()
	if err != nil {
		values = append(values, []string{
			*d.DeploymentConfig.Name,
			"Error",
			*d.DeploymentConfig.Namespace,
			err.Error(),
		})

		return values, nil
	}

	if releases == nil || len(releases.Releases) == 0 {
		values = append(values, []string{
			*d.DeploymentConfig.Name,
			"Not Found",
			*d.DeploymentConfig.Namespace,
			"No release found",
		})

		return values, nil
	}

	for _, release := range releases.Releases {
		if release.GetName() == *d.DeploymentConfig.Name {
			if release.Info.Status.Code.String() != "DEPLOYED" {
				values = append(values, []string{
					*d.DeploymentConfig.Name,
					"Error",
					*d.DeploymentConfig.Namespace,
					"HELM STATUS:" + release.Info.Status.Code.String(),
				})

				return values, nil
			}

			values = append(values, []string{
				*d.DeploymentConfig.Name,
				"Running",
				*d.DeploymentConfig.Namespace,
				"",
			})

			return values, nil
		}
	}

	values = append(values, []string{
		*d.DeploymentConfig.Name,
		"Not Found",
		*d.DeploymentConfig.Namespace,
		"No release found",
	})

	return values, nil
}

// Deploy deploys the given deployment with helm
func (d *DeployConfig) Deploy(generatedConfig *generated.Config, forceDeploy bool) error {
	var overrideTimestamp int64
	var overridePath string

	releaseName := *d.DeploymentConfig.Name
	chartPath := *d.DeploymentConfig.Helm.ChartPath

	// Check if the chart directory has changed
	hash, err := hash.Directory(chartPath)
	if err != nil {
		return fmt.Errorf("Error hashing chart directory: %v", err)
	}

	if d.DeploymentConfig.Helm.Override != nil {
		stat, err := os.Stat(*d.DeploymentConfig.Helm.Override)
		if err != nil {
			return fmt.Errorf("Error stating override file: %s", *d.DeploymentConfig.Helm.Override)
		}

		overridePath = *d.DeploymentConfig.Helm.Override
		overrideTimestamp = stat.ModTime().Unix()
	}

	// Get HelmClient
	helmClient, err := helm.NewClient(d.KubeClient, d.Log, false)
	if err != nil {
		return fmt.Errorf("Error creating helm client: %v", err)
	}

	// Check if redeploying is necessary
	reDeploy := forceDeploy || generatedConfig.HelmChartHashs[chartPath] != hash || (overrideTimestamp != 0 && generatedConfig.HelmOverrideTimestamps[overridePath] != overrideTimestamp)
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
		err = d.internalDeploy(generatedConfig, helmClient)
		if err != nil {
			return err
		}

		generatedConfig.HelmChartHashs[chartPath] = hash

		if overrideTimestamp != 0 {
			generatedConfig.HelmOverrideTimestamps[overridePath] = overrideTimestamp
		}
	} else {
		d.Log.Infof("Skipping chart %s", chartPath)
	}

	return nil
}

func (d *DeployConfig) internalDeploy(generatedConfig *generated.Config, helmClient *helm.ClientWrapper) error {
	d.Log.StartWait("Deploying helm chart")
	defer d.Log.StopWait()

	config := configutil.GetConfig()

	releaseName := *d.DeploymentConfig.Name
	releaseNamespace := *d.DeploymentConfig.Namespace
	chartPath := *d.DeploymentConfig.Helm.ChartPath

	values := map[interface{}]interface{}{}
	overwriteValues := map[interface{}]interface{}{}

	valuesPath := filepath.Join(chartPath, "values.yaml")
	err := yamlutil.ReadYamlFromFile(valuesPath, values)
	if err != nil {
		return fmt.Errorf("Couldn't deploy chart, error reading from chart values %s: %v", valuesPath, err)
	}

	// Load override values from path
	if d.DeploymentConfig.Helm.Override != nil {
		overwriteValuesPath, err := filepath.Abs(*d.DeploymentConfig.Helm.Override)
		if err != nil {
			return fmt.Errorf("Error retrieving absolute path from %s: %v", *d.DeploymentConfig.Helm.Override, err)
		}

		err = yamlutil.ReadYamlFromFile(overwriteValuesPath, overwriteValues)
		if err != nil {
			d.Log.Warnf("Error reading from chart dev overwrite values %s: %v", overwriteValuesPath, err)
		}
	} else if d.UseDevOverwrite && d.DeploymentConfig.Helm.DevOverwrite != nil {
		overwriteValuesPath, err := filepath.Abs(*d.DeploymentConfig.Helm.DevOverwrite)
		if err != nil {
			return fmt.Errorf("Error retrieving absolute path from %s: %v", *d.DeploymentConfig.Helm.DevOverwrite, err)
		}

		err = yamlutil.ReadYamlFromFile(overwriteValuesPath, overwriteValues)
		if err != nil {
			d.Log.Warnf("Error reading from chart dev overwrite values %s: %v", overwriteValuesPath, err)
		}
	}

	// Load override values from data and merge them
	if d.DeploymentConfig.Helm.OverrideValues != nil {
		Values(overwriteValues).MergeInto(*d.DeploymentConfig.Helm.OverrideValues)
	}

	// Set containers and pull secrets values
	overwriteValues["containers"] = getContainerValues(overwriteValues, config, generatedConfig)
	overwriteValues["pullSecrets"] = getPullSecrets(values, overwriteValues, config)

	wait := true
	if d.DeploymentConfig.Helm.Wait != nil && *d.DeploymentConfig.Helm.Wait == false {
		wait = *d.DeploymentConfig.Helm.Wait
	}

	appRelease, err := helmClient.InstallChartByPath(releaseName, releaseNamespace, chartPath, &overwriteValues, wait)
	if err != nil {
		return fmt.Errorf("Unable to deploy helm chart: %v", err)
	}

	releaseRevision := int(appRelease.Version)
	d.Log.Donef("Deployed helm chart (Release revision: %d)", releaseRevision)

	return nil
}

func getContainerValues(overwriteValues map[interface{}]interface{}, config *v1.Config, generatedConfig *generated.Config) map[interface{}]interface{} {
	overwriteContainerValues := map[interface{}]interface{}{}
	overwriteContainerValuesFromFile, containerValuesExisting := overwriteValues["containers"]
	if containerValuesExisting {
		overwriteContainerValues = overwriteContainerValuesFromFile.(map[interface{}]interface{})
	}

	for imageName, imageConf := range *config.Images {
		container := map[interface{}]interface{}{}
		existingContainer, containerExists := overwriteContainerValues[imageName]

		if containerExists {
			container = existingContainer.(map[interface{}]interface{})
		}
		container["image"] = registry.GetImageURL(generatedConfig, imageConf, true)

		overwriteContainerValues[imageName] = container
	}

	return overwriteContainerValues
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

	for _, registryConf := range *config.Registries {
		if registryConf.URL != nil {
			registrySecretName := registry.GetRegistryAuthSecretName(*registryConf.URL)
			overwritePullSecrets = append(overwritePullSecrets, registrySecretName)
		}
	}

	for _, autoGeneratedPullSecret := range registry.GetPullSecretNames() {
		overwritePullSecrets = append(overwritePullSecrets, autoGeneratedPullSecret)
	}

	return overwritePullSecrets
}
