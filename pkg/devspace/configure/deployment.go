package configure

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	dockerfileutil "github.com/devspace-cloud/devspace/pkg/util/dockerfile"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/pkg/errors"
)

var imageNameCleaningRegex = regexp.MustCompile("[^a-z0-9]")

// GetDockerfileComponentDeployment returns a new deployment that deploys an image built from a local dockerfile via a component
func GetDockerfileComponentDeployment(config *latest.Config, generatedConfig *generated.Config, name, imageName, dockerfile, context string) (*latest.ImageConfig, *latest.DeploymentConfig, error) {
	var imageConfig *latest.ImageConfig
	var err error
	if imageName == "" {
		imageName = imageNameCleaningRegex.ReplaceAllString(strings.ToLower(name), "")
		imageConfig, err = GetImageConfigFromDockerfile(config, imageName, dockerfile, context)
		if err != nil {
			return nil, nil, errors.Wrap(err, "get image config")
		}
		imageName = imageConfig.Image
	} else {
		imageConfig = GetImageConfigFromImageName(imageName, dockerfile, context)
	}

	// Prepare return deployment config
	retDeploymentConfig := &latest.DeploymentConfig{
		Name: name,
		Component: &latest.ComponentConfig{
			Containers: []*latest.ContainerConfig{
				{
					Image: imageName,
				},
			},
		},
	}

	// Try to get ports from dockerfile
	port := ""
	ports, err := dockerfileutil.GetPorts(dockerfile)
	if err == nil {
		if len(ports) == 1 {
			port = strconv.Itoa(ports[0])
		} else if len(ports) > 1 {
			port = survey.Question(&survey.QuestionOptions{
				Question:     "Which port is your application listening on?",
				DefaultValue: strconv.Itoa(ports[0]),
			})
			if port == "" {
				port = strconv.Itoa(ports[0])
			}
		}
	}
	if port == "" {
		port = survey.Question(&survey.QuestionOptions{
			Question: "Which port is your application listening on? (Enter to skip)",
		})
	}
	if port != "" {
		port, err := strconv.Atoi(port)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing port")
		}

		retDeploymentConfig.Component.Service = &latest.ServiceConfig{
			Ports: []*latest.ServicePortConfig{
				{
					Port: &port,
				},
			},
		}
	}

	return imageConfig, retDeploymentConfig, nil
}

// GetImageComponentDeployment returns a new deployment that deploys an image via a component
func GetImageComponentDeployment(name, imageName string) (*latest.ImageConfig, *latest.DeploymentConfig, error) {
	retDeploymentConfig := &latest.DeploymentConfig{
		Name: name,
		Component: &latest.ComponentConfig{
			Containers: []*latest.ContainerConfig{
				{
					Image: imageName,
				},
			},
		},
	}

	// Configure port
	port := survey.Question(&survey.QuestionOptions{
		Question: "Which port do you want to expose for this image? (Enter to skip)",
	})
	if port != "" {
		port, err := strconv.Atoi(port)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing port")
		}

		retDeploymentConfig.Component.Service = &latest.ServiceConfig{
			Ports: []*latest.ServicePortConfig{
				{
					Port: &port,
				},
			},
		}
	}

	// Check if we should create pull secret
	retImageConfig := GetImageConfigFromImageName(imageName, "", "")
	return retImageConfig, retDeploymentConfig, nil
}

// GetPredefinedComponentDeployment returns deployment that uses a predefined component
func GetPredefinedComponentDeployment(name, component string) (*latest.DeploymentConfig, error) {
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
		Name:      name,
		Component: template,
	}, nil
}

// GetKubectlDeployment retruns a new kubectl deployment
func GetKubectlDeployment(name, manifests string) (*latest.DeploymentConfig, error) {
	splitted := strings.Split(manifests, ",")
	splittedPointer := []string{}

	for _, s := range splitted {
		trimmed := strings.TrimSpace(s)
		splittedPointer = append(splittedPointer, trimmed)
	}

	return &v1.DeploymentConfig{
		Name: name,
		Kubectl: &v1.KubectlConfig{
			Manifests: splittedPointer,
		},
	}, nil
}

// GetHelmDeployment returns a new helm deployment
func GetHelmDeployment(name, chartName, chartRepo, chartVersion string) (*latest.DeploymentConfig, error) {
	retDeploymentConfig := &v1.DeploymentConfig{
		Name: name,
		Helm: &v1.HelmConfig{
			Chart: &v1.ChartConfig{
				Name: chartName,
			},
		},
	}

	if chartRepo != "" {
		retDeploymentConfig.Helm.Chart.RepoURL = chartRepo
	}
	if chartVersion != "" {
		retDeploymentConfig.Helm.Chart.Version = chartVersion
	}

	return retDeploymentConfig, nil
}

// RemoveDeployment removes one or all deployments from the config
func RemoveDeployment(removeAll bool, name string) (bool, error) {
	if name == "" && removeAll == false {
		return false, errors.New("You have to specify either a deployment name or the --all flag")
	}

	config := configutil.GetBaseConfig(context.Background())
	found := false

	if config.Deployments != nil {
		newDeployments := []*v1.DeploymentConfig{}

		for _, deployConfig := range config.Deployments {
			if removeAll == false && deployConfig.Name != name {
				newDeployments = append(newDeployments, deployConfig)
			} else {
				found = true
			}
		}

		config.Deployments = newDeployments
	}

	err := configutil.SaveLoadedConfig()
	if err != nil {
		return false, fmt.Errorf("Couldn't save config file: %s", err.Error())
	}

	return found, nil
}
