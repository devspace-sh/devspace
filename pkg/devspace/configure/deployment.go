package configure

import (
	"errors"
	"fmt"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
)

// AddDeployment adds a new deployment to the config
func AddDeployment(name, namespace, manifests, chart string) error {
	if manifests == "" && chart == "" {
		return errors.New("Either manifests or chart flag has to be specified")
	}
	if manifests != "" && chart != "" {
		return errors.New("The --manifests flag and --chart flag cannot be used together")
	}

	config := configutil.GetBaseConfig()

	if config.Deployments != nil {
		for _, deployConfig := range *config.Deployments {
			if *deployConfig.Name == name {
				return fmt.Errorf("Deployment %s already exists", name)
			}
		}
	} else {
		config.Deployments = &[]*v1.DeploymentConfig{}
	}

	deployments := *config.Deployments

	if chart != "" {
		deployments = append(deployments, &v1.DeploymentConfig{
			Name:      &name,
			Namespace: &namespace,
			Helm: &v1.HelmConfig{
				ChartPath: &chart,
			},
		})
	} else if manifests != "" {
		splitted := strings.Split(manifests, ",")
		splittedPointer := []*string{}

		for _, s := range splitted {
			s = strings.TrimSpace(s)
			splittedPointer = append(splittedPointer, &s)
		}

		deployments = append(deployments, &v1.DeploymentConfig{
			Name:      &name,
			Namespace: &namespace,
			Kubectl: &v1.KubectlConfig{
				Manifests: &splittedPointer,
			},
		})
	}

	config.Deployments = &deployments

	err := configutil.SaveBaseConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %s", err.Error())
	}

	return nil
}

// RemoveDeployment removes one or all deployments from the config
func RemoveDeployment(removeAll bool, name string) error {
	if name == "" && removeAll == false {
		return errors.New("You have to specify either a deployment name or the --all flag")
	}

	config := configutil.GetBaseConfig()

	if config.Deployments != nil {
		newDeployments := []*v1.DeploymentConfig{}

		for _, deployConfig := range *config.Deployments {
			if removeAll == false && *deployConfig.Name != name {
				newDeployments = append(newDeployments, deployConfig)
			}
		}

		config.Deployments = &newDeployments
	}

	err := configutil.SaveBaseConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %s", err.Error())
	}

	return nil
}
