package configure

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	dockerfileutil "github.com/devspace-cloud/devspace/pkg/util/dockerfile"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
)

var imageNameCleaningRegex = regexp.MustCompile("[^a-z0-9]")

// NewDockerfileComponentDeployment returns a new deployment that deploys an image built from a local dockerfile via a component
func (m *manager) NewDockerfileComponentDeployment(generatedConfig *generated.Config, name, imageName, dockerfile, context string) (*latest.ImageConfig, *latest.DeploymentConfig, error) {
	var imageConfig *latest.ImageConfig
	var err error
	if imageName == "" {
		imageName = imageNameCleaningRegex.ReplaceAllString(strings.ToLower(name), "")
		imageConfig, err = m.newImageConfigFromDockerfile(imageName, dockerfile, context)
		if err != nil {
			return nil, nil, errors.Wrap(err, "get image config")
		}
		imageName = imageConfig.Image
	} else {
		imageConfig = m.newImageConfigFromImageName(imageName, dockerfile, context)
	}

	componentConfig := &latest.ComponentConfig{
		Containers: []*latest.ContainerConfig{
			{
				Image: imageName,
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
			port, err = m.log.Question(&survey.QuestionOptions{
				Question:     "Which port is your application listening on?",
				DefaultValue: strconv.Itoa(ports[0]),
			})
			if err != nil {
				return nil, nil, err
			}

			if port == "" {
				port = strconv.Itoa(ports[0])
			}
		}
	}
	if port == "" {
		port, err = m.log.Question(&survey.QuestionOptions{
			Question: "Which port is your application listening on? (Enter to skip)",
		})
		if err != nil {
			return nil, nil, err
		}
	}
	if port != "" {
		port, err := strconv.Atoi(port)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing port")
		}

		componentConfig.Service = &latest.ServiceConfig{
			Ports: []*latest.ServicePortConfig{
				{
					Port: &port,
				},
			},
		}
	}

	retDeploymentConfig, err := generateComponentDeployment(name, componentConfig)
	if err != nil {
		return nil, nil, err
	}

	return imageConfig, retDeploymentConfig, nil
}

// NewImageComponentDeployment returns a new deployment that deploys an image via a component
func (m *manager) NewImageComponentDeployment(name, imageName string) (*latest.ImageConfig, *latest.DeploymentConfig, error) {
	componentConfig := &latest.ComponentConfig{
		Containers: []*latest.ContainerConfig{
			{
				Image: imageName,
			},
		},
	}

	// Configure port
	port, err := m.log.Question(&survey.QuestionOptions{
		Question: "Which port do you want to expose for this image? (Enter to skip)",
	})
	if err != nil {
		return nil, nil, err
	}
	if port != "" {
		port, err := strconv.Atoi(port)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing port")
		}

		componentConfig.Service = &latest.ServiceConfig{
			Ports: []*latest.ServicePortConfig{
				{
					Port: &port,
				},
			},
		}
	}

	retDeploymentConfig, err := generateComponentDeployment(name, componentConfig)
	if err != nil {
		return nil, nil, err
	}

	// Check if we should create pull secret
	retImageConfig := m.newImageConfigFromImageName(imageName, "", "")
	return retImageConfig, retDeploymentConfig, nil
}

func generateComponentDeployment(name string, componentConfig *latest.ComponentConfig) (*latest.DeploymentConfig, error) {
	chartValues, err := yamlutil.ToInterfaceMap(componentConfig)
	if err != nil {
		return nil, err
	}

	// Prepare return deployment config
	retDeploymentConfig := &latest.DeploymentConfig{
		Name: name,
		Helm: &latest.HelmConfig{
			ComponentChart: ptr.Bool(true),
			Values:         chartValues,
		},
	}
	return retDeploymentConfig, nil
}

// NewKubectlDeployment retruns a new kubectl deployment
func (m *manager) NewKubectlDeployment(name, manifests string) (*latest.DeploymentConfig, error) {
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

// NewHelmDeployment returns a new helm deployment
func (m *manager) NewHelmDeployment(name, chartName, chartRepo, chartVersion string) (*latest.DeploymentConfig, error) {
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
func (m *manager) RemoveDeployment(removeAll bool, name string) (bool, error) {
	if name == "" && removeAll == false {
		return false, errors.New("You have to specify either a deployment name or the --all flag")
	}

	found := false

	if m.config.Deployments != nil {
		newDeployments := []*v1.DeploymentConfig{}

		for _, deployConfig := range m.config.Deployments {
			if removeAll == false && deployConfig.Name != name {
				newDeployments = append(newDeployments, deployConfig)
			} else {
				found = true
			}
		}

		m.config.Deployments = newDeployments
	}

	return found, nil
}
