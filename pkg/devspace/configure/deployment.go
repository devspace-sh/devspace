package configure

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"
	"github.com/pkg/errors"
)

// GetImageComponentDeployment adds a new deployment that deploys an image via a component
func GetImageComponentDeployment(config *latest.Config, name, imageName string) (*latest.DeploymentConfig, error) {
	retDeploymentConfig := &latest.DeploymentConfig{
		Name: &name,
		Component: &latest.ComponentConfig{
			Containers: &[]*latest.ContainerConfig{
				{
					Image: &imageName,
				},
			},
		},
	}

	// Configure port
	port := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question: "Which port do you want to expose for this image? (Enter to skip)",
	})
	if port != "" {
		port, err := strconv.Atoi(port)
		if err != nil {
			return nil, errors.Wrap(err, "parsing port")
		}

		retDeploymentConfig.Component.Service = &latest.ServiceConfig{
			Ports: &[]*latest.ServicePortConfig{
				{
					Port: &port,
				},
			},
		}
	}

	// Configure pull secret
	createPullSecret := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question: "Do you want to enable automatic creation of pull secrets for this image?",
		Options:  []string{"yes", "no"},
	}) == "yes"
	if createPullSecret {
		// Figure out tag
		splittedImage := strings.Split(imageName, ":")
		imageTag := "latest"
		if len(splittedImage) > 1 {
			imageTag = splittedImage[1]
		}

		// Add to image config
		(*config.Images)[name] = &latest.ImageConfig{
			Image:            &splittedImage[0],
			Tag:              &imageTag,
			CreatePullSecret: &createPullSecret,
			Build: &latest.BuildConfig{
				Disabled: ptr.Bool(true),
			},
		}
	}

	return retDeploymentConfig, nil
}

// GetPredefinedComponentDeployment returns deployment that uses a predefined component
func GetPredefinedComponentDeployment(config *latest.Config, name, component string) (*latest.DeploymentConfig, error) {
	// Create component generator
	componentGenerator, err := generator.NewComponentGenerator()
	if err != nil {
		return nil, fmt.Errorf("Error initializing component generator: %v", err)
	}

	// Get component template
	template, err := componentGenerator.GetComponentTemplate(component)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving template: %v", err)
	}

	return &latest.DeploymentConfig{
		Name:      &name,
		Component: template,
	}, nil
}

// GetKubectlDeployment retruns a new kubectl deployment
func GetKubectlDeployment(config *latest.Config, name, manifests string) (*latest.DeploymentConfig, error) {
	splitted := strings.Split(manifests, ",")
	splittedPointer := []*string{}

	for _, s := range splitted {
		s = strings.TrimSpace(s)
		splittedPointer = append(splittedPointer, &s)
	}

	return &v1.DeploymentConfig{
		Name: &name,
		Kubectl: &v1.KubectlConfig{
			Manifests: &splittedPointer,
		},
	}, nil
}

// GetHelmDeployment returns a new helm deployment
func GetHelmDeployment(config *latest.Config, name, chartName, chartRepo, chartVersion string) (*latest.DeploymentConfig, error) {
	retDeploymentConfig := &v1.DeploymentConfig{
		Name: &name,
		Helm: &v1.HelmConfig{
			Chart: &v1.ChartConfig{
				Name: &chartName,
			},
		},
	}

	if chartRepo != "" {
		retDeploymentConfig.Helm.Chart.RepoURL = &chartRepo
	}
	if chartVersion != "" {
		retDeploymentConfig.Helm.Chart.Version = &chartVersion
	}

	return retDeploymentConfig, nil
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
