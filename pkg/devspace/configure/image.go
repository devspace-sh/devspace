package configure

import (
	contextpkg "context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/builder/helper"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/pkg/errors"
)

const dockerHubHostname = "hub.docker.com"

// GetImageConfigFromImageName returns an image config based on the image
func GetImageConfigFromImageName(imageName, dockerfile, context string) *latest.ImageConfig {
	// Figure out tag
	imageTag := ""
	splittedImage := strings.Split(imageName, ":")
	imageName = splittedImage[0]
	if len(splittedImage) > 1 {
		imageTag = splittedImage[1]
	} else if dockerfile == "" {
		imageTag = "latest"
	}

	retImageConfig := &latest.ImageConfig{
		Image:            imageName,
		CreatePullSecret: ptr.Bool(true),
	}

	if imageTag != "" {
		retImageConfig.Tag = imageTag
	}
	if dockerfile == "" {
		retImageConfig.Build = &latest.BuildConfig{
			Disabled: ptr.Bool(true),
		}
	} else {
		if dockerfile != helper.DefaultDockerfilePath {
			retImageConfig.Dockerfile = dockerfile
		}
		if context != "" && context != helper.DefaultContextPath {
			retImageConfig.Context = context
		}
	}

	return retImageConfig
}

// GetImageConfigFromDockerfile gets the image config based on the configured cloud provider or asks the user where to push to
func GetImageConfigFromDockerfile(config *latest.Config, imageName, dockerfile, context string, log log.Logger) (*latest.ImageConfig, error) {
	var (
		dockerUsername = ""
		retImageConfig = &latest.ImageConfig{}
	)

	// Ignore error as context may not be a Space
	kubeContext, err := kubeconfig.GetCurrentContext()
	if err != nil {
		return nil, err
	}

	// Get docker client
	client, err := docker.NewClientWithMinikube(kubeContext, true, log)
	if err != nil {
		return nil, errors.Errorf("Cannot create docker client: %v", err)
	}

	// Check if docker is installed
	_, err = client.Ping(contextpkg.Background())
	if err != nil {
		// Check if docker cli is installed
		runErr := exec.Command("docker").Run()
		if runErr == nil {
			log.Warn("Docker daemon not running. Start Docker daemon to build images with Docker instead of using the kaniko fallback.")
		}
	}

	// Get cloud provider if context is a space
	cloudProvider, err := cloud.GetDefaultProviderName()
	if err != nil {
		return nil, err
	}

	cloudRegistryHostname, err := getCloudRegistryHostname(&cloudProvider)
	if err != nil {
		return nil, err
	}

	registryURL, err := getRegistryURL(config, cloudRegistryHostname, &cloudProvider, log)
	if err != nil {
		return nil, err
	}

	if registryURL == cloudRegistryHostname {
		imageName = registryURL + "/${DEVSPACE_USERNAME}/" + imageName
	} else if registryURL == "hub.docker.com" {
		log.StartWait("Checking Docker credentials")
		dockerAuthConfig, err := client.GetAuthConfig("", true)
		log.StopWait()
		if err == nil {
			dockerUsername = dockerAuthConfig.Username
		}

		imageName, err = survey.Question(&survey.QuestionOptions{
			Question:          "Which image name do you want to use on Docker Hub?",
			DefaultValue:      dockerUsername + "/" + imageName,
			ValidationMessage: "Please enter a valid image name for Docker Hub (e.g. myregistry.com/user/repository | allowed charaters: /, a-z, 0-9)",
			ValidationFunc: func(name string) error {
				_, err := registry.GetStrippedDockerImageName(name)
				return err
			},
		}, log)
		if err != nil {
			return nil, err
		}

		imageName, _ = registry.GetStrippedDockerImageName(imageName)
	} else if regexp.MustCompile("^(.+\\.)?gcr.io$").Match([]byte(registryURL)) { // Is google registry?
		project, err := exec.Command("gcloud", "config", "get-value", "project").Output()
		gcloudProject := "myGCloudProject"

		if err == nil {
			gcloudProject = strings.TrimSpace(string(project))
		}

		imageName, err = survey.Question(&survey.QuestionOptions{
			Question:          "Which image name do you want to push to?",
			DefaultValue:      registryURL + "/" + gcloudProject + "/" + imageName,
			ValidationMessage: "Please enter a valid Docker image name (e.g. myregistry.com/user/repository | allowed charaters: /, a-z, 0-9)",
			ValidationFunc: func(name string) error {
				_, err := registry.GetStrippedDockerImageName(name)
				return err
			},
		}, log)
		if err != nil {
			return nil, err
		}

		imageName, _ = registry.GetStrippedDockerImageName(imageName)
	} else {
		if dockerUsername == "" {
			dockerUsername = "myuser"
		}

		imageName, err = survey.Question(&survey.QuestionOptions{
			Question:          "Which image name do you want to push to?",
			DefaultValue:      registryURL + "/" + dockerUsername + "/" + imageName,
			ValidationMessage: "Please enter a valid docker image name (e.g. myregistry.com/user/repository)",
			ValidationFunc: func(name string) error {
				_, err := registry.GetStrippedDockerImageName(name)
				return err
			},
		}, log)
		if err != nil {
			return nil, err
		}

		imageName, _ = registry.GetStrippedDockerImageName(imageName)
	}

	// Set image name
	retImageConfig.Image = imageName

	// Set image specifics
	if dockerfile != "" && dockerfile != helper.DefaultDockerfilePath {
		retImageConfig.Dockerfile = dockerfile
	}
	if context != "" && context != helper.DefaultContextPath {
		retImageConfig.Context = context
	}

	return retImageConfig, nil
}

func getRegistryURL(config *latest.Config, cloudRegistryHostname string, cloudProvider *string, log log.Logger) (string, error) {
	var (
		useDockerHub          = "Use " + dockerHubHostname
		useDevSpaceRegistry   = "Use " + cloudRegistryHostname + " (free, private Docker registry)"
		useOtherRegistry      = "Use other registry"
		registryUsernameHint  = " => you are logged in as %s"
		registryDefaultOption = useDevSpaceRegistry
		registryLoginHint     = "Please login via `docker login%s` and try again."
	)

	// Initialize docker
	dockerClient, err := docker.NewClient(log)
	if err != nil {
		return "", errors.Errorf("Error creating docker client: %v", err)
	}

	authConfig, err := dockerClient.GetAuthConfig(dockerHubHostname, true)
	if err == nil && authConfig.Username != "" {
		useDockerHub = useDockerHub + fmt.Sprintf(registryUsernameHint, authConfig.Username)
		registryDefaultOption = useDockerHub
	}

	registryOptions := []string{useDockerHub, useOtherRegistry}
	if cloudRegistryHostname != "" {
		authConfig, err = dockerClient.GetAuthConfig(cloudRegistryHostname, true)
		if err == nil && authConfig.Username != "" {
			useDevSpaceRegistry = useDevSpaceRegistry + fmt.Sprintf(registryUsernameHint, authConfig.Username)
			registryDefaultOption = useDevSpaceRegistry
		}

		registryOptions = []string{useDockerHub, useDevSpaceRegistry, useOtherRegistry}
	}

	selectedRegistry, err := survey.Question(&survey.QuestionOptions{
		Question:     "Which registry do you want to use for storing your Docker images?",
		DefaultValue: registryDefaultOption,
		Options:      registryOptions,
	}, log)
	if err != nil {
		return "", err
	}

	var registryURL string
	if selectedRegistry == useDockerHub {
		registryURL = dockerHubHostname
	} else if selectedRegistry == useDevSpaceRegistry {
		registryURL = cloudRegistryHostname
		registryLoginHint = fmt.Sprintf(registryLoginHint, " "+cloudRegistryHostname)
	} else {
		registryURL, err = survey.Question(&survey.QuestionOptions{
			Question:     "Please enter the registry URL without image name:",
			DefaultValue: "my.registry.tld/username",
		}, log)
		if err != nil {
			return "", err
		}

		registryURL = strings.Trim(registryURL, "/ ")
		registryLoginHint = fmt.Sprintf(registryLoginHint, " "+registryURL)
	}

	log.StartWait("Checking registry authentication")
	authConfig, err = dockerClient.Login(registryURL, "", "", true, false, false)
	log.StopWait()
	if err != nil || authConfig.Username == "" {
		if registryURL == dockerHubHostname {
			log.Warn("You are not logged in to Docker Hub")
			log.Warn("Please make sure you have a https://hub.docker.com account")
			log.Warn("Installing docker is NOT required. You simply need a Docker Hub account\n")

			for {
				dockerUsername, err := survey.Question(&survey.QuestionOptions{
					Question:               "What is your Docker Hub username?",
					DefaultValue:           "",
					ValidationRegexPattern: "^.*$",
				}, log)
				if err != nil {
					return "", err
				}

				dockerPassword, err := survey.Question(&survey.QuestionOptions{
					Question:               "What is your Docker Hub password? (will only be sent to Docker Hub)",
					DefaultValue:           "",
					ValidationRegexPattern: "^.*$",
					IsPassword:             true,
				}, log)
				if err != nil {
					return "", err
				}

				_, err = dockerClient.Login(registryURL, dockerUsername, dockerPassword, false, true, true)
				if err != nil {
					log.Warn(err)
					continue
				}

				break
			}
		} else if selectedRegistry == useDevSpaceRegistry {
			return registryURL, loginDevSpaceCloud(*cloudProvider, log)
		} else {
			return "", errors.Errorf("Registry authentication failed for %s.\n         %s", registryURL, registryLoginHint)
		}
	}

	return registryURL, nil
}

func getCloudRegistryHostname(cloudProvider *string) (string, error) {
	var registryURL string

	if cloudProvider == nil || *cloudProvider == "" || *cloudProvider == cloudconfig.DevSpaceCloudProviderName {
		// prevents EnsureLoggedIn call in GetProvider
		// TODO: remove this hard-coded exception once the registry URL can be retrieved from DevSpace Cloud without login
		registryURL = "dscr.io"
	} else {
		// Get default registry
		provider, err := cloud.GetProvider(ptr.ReverseString(cloudProvider), log.GetInstance())
		if err != nil {
			return "", errors.Errorf("Error login into cloud provider: %v", err)
		}

		registries, err := provider.GetRegistries()
		if err != nil {
			return "", errors.Errorf("Error retrieving registries: %v", err)
		}
		if len(registries) > 0 {
			registryURL = registries[0].URL
		}
	}

	return registryURL, nil
}

func loginDevSpaceCloud(cloudProvider string, log log.Logger) error {
	// Ensure user is logged in
	_, err := cloud.GetProvider(cloudProvider, log)
	return err
}

// AddImage adds an image to the devspace
func AddImage(baseConfig *latest.Config, nameInConfig, name, tag, contextPath, dockerfilePath, buildTool string) error {
	imageConfig := &v1.ImageConfig{
		Image: name,
	}

	if tag != "" {
		imageConfig.Tag = tag
	}
	if contextPath != "" {
		imageConfig.Context = contextPath
	}
	if dockerfilePath != "" {
		imageConfig.Dockerfile = dockerfilePath
	}

	if buildTool == "docker" {
		if imageConfig.Build == nil {
			imageConfig.Build = &v1.BuildConfig{}
		}

		imageConfig.Build.Docker = &v1.DockerConfig{}
	} else if buildTool == "kaniko" {
		if imageConfig.Build == nil {
			imageConfig.Build = &v1.BuildConfig{}
		}

		imageConfig.Build.Kaniko = &v1.KanikoConfig{}
	} else if buildTool != "" {
		log.Errorf("BuildTool %v unknown. Please select one of docker|kaniko", buildTool)
	}

	if baseConfig.Images == nil {
		images := make(map[string]*v1.ImageConfig)
		baseConfig.Images = images
	}

	baseConfig.Images[nameInConfig] = imageConfig

	err := configutil.SaveLoadedConfig()
	if err != nil {
		return errors.Errorf("Couldn't save config file: %s", err.Error())
	}

	return nil
}

//RemoveImage removes an image from the devspace
func RemoveImage(baseConfig *latest.Config, removeAll bool, names []string) error {
	if len(names) == 0 && removeAll == false {
		return errors.Errorf("You have to specify at least one image")
	}

	newImageList := make(map[string]*v1.ImageConfig)

	if !removeAll && baseConfig.Images != nil {
	ImagesLoop:
		for nameInConfig, imageConfig := range baseConfig.Images {
			for _, deletionName := range names {
				if deletionName == nameInConfig {
					continue ImagesLoop
				}
			}

			newImageList[nameInConfig] = imageConfig
		}
	}

	baseConfig.Images = newImageList

	err := configutil.SaveLoadedConfig()
	if err != nil {
		return errors.Errorf("Couldn't save config file: %v", err)
	}

	return nil
}
