package component

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/helm"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"k8s.io/client-go/kubernetes"
)

// DeployConfig holds the informations for deploying a component
type DeployConfig struct {
	HelmConfig *helm.DeployConfig
}

// DevSpaceChartConfig is the config that holds the devspace chart information
var DevSpaceChartConfig = &latest.ChartConfig{
	Name:    ptr.String("component-chart"),
	Version: ptr.String("v0.0.1"),
	RepoURL: ptr.String("https://charts.devspace.cloud"),
}

// New creates a new helm deployment client
func New(kubectl kubernetes.Interface, deployConfig *latest.DeploymentConfig, log log.Logger) (*DeployConfig, error) {
	// Convert the values
	values := map[interface{}]interface{}{}
	err := util.Convert(deployConfig.Component, &values)
	if err != nil {
		return nil, err
	}

	// Create a helm config out of the deployment config
	helmConfig, err := helm.New(kubectl, &latest.DeploymentConfig{
		Name:      deployConfig.Name,
		Namespace: deployConfig.Namespace,
		Helm: &latest.HelmConfig{
			Chart:  DevSpaceChartConfig,
			Values: &values,
		},
	}, log)
	if err != nil {
		return nil, err
	}

	return &DeployConfig{
		HelmConfig: helmConfig,
	}, nil
}

// Deploy deploys the given deployment with helm
func (d *DeployConfig) Deploy(cache *generated.CacheConfig, forceDeploy bool, builtImages map[string]string) (bool, error) {
	return d.HelmConfig.Deploy(cache, forceDeploy, builtImages)
}

// Status gets the status of the deployment
func (d *DeployConfig) Status() (*deploy.StatusResult, error) {
	status, err := d.HelmConfig.Status()
	if err != nil {
		return nil, err
	}

	status.Type = "Component"
	return status, nil
}

// Delete deletes the release
func (d *DeployConfig) Delete(cache *generated.CacheConfig) error {
	return d.HelmConfig.Delete(cache)
}
