package component

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/helm"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"k8s.io/client-go/kubernetes"
)

// DeployConfig holds the informations for deploying a component
type DeployConfig struct {
	helmConfig *helm.DeployConfig
}

// DevSpaceChartConfig is the config that holds the devspace chart information
var DevSpaceChartConfig = &latest.ChartConfig{
	Name:    ptr.String("component-chart"),
	Version: ptr.String("v0.0.1"),
	RepoURL: ptr.String("https://charts.devspace.cloud"),
}

// New creates a new helm deployment client
func New(kubectl *kubernetes.Clientset, deployConfig *latest.DeploymentConfig, log log.Logger) (*DeployConfig, error) {
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
		helmConfig: helmConfig,
	}, nil
}

// Deploy deploys the given deployment with helm
func (d *DeployConfig) Deploy(generatedConfig *generated.Config, isDev, forceDeploy bool) error {
	return d.helmConfig.Deploy(generatedConfig, isDev, forceDeploy)
}

// Status gets the status of the deployment
func (d *DeployConfig) Status() ([][]string, error) {
	return d.helmConfig.Status()
}

// Delete deletes the release
func (d *DeployConfig) Delete() error {
	return d.helmConfig.Delete()
}
