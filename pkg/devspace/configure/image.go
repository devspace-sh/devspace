package configure

import (
	contextpkg "context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/image"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/minikube"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"
)

// DefaultImageName is the default image name
const DefaultImageName = "devspace"

// GetImageConfigFromImageName returns an image config based on the image
func GetImageConfigFromImageName(imageName, dockerfile, context string) *latest.ImageConfig {
	// Configure pull secret
	createPullSecret := dockerfile != "" || *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question: "Do you want to enable automatic creation of pull secrets for this image?",
		Options:  []string{"no", "yes"},
	}) == "yes"

	if createPullSecret {
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
			Image:            &imageName,
			CreatePullSecret: &createPullSecret,
		}

		if imageTag != "" {
			retImageConfig.Tag = &imageTag
		}
		if dockerfile == "" {
			retImageConfig.Build = &latest.BuildConfig{
				Disabled: ptr.Bool(true),
			}
		} else {
			if dockerfile != image.DefaultDockerfilePath {
				if retImageConfig.Build == nil {
					retImageConfig.Build = &latest.BuildConfig{}
				}
				retImageConfig.Build.Dockerfile = &dockerfile
			}
			if context != "" && context != image.DefaultContextPath {
				if retImageConfig.Build == nil {
					retImageConfig.Build = &latest.BuildConfig{}
				}
				retImageConfig.Build.Context = &context
			}
		}

		return retImageConfig
	}

	return nil
}

// GetImageConfigFromDockerfile gets the image config based on the configured cloud provider or asks the user where to push to
func GetImageConfigFromDockerfile(dockerfile, context string, cloudProvider *string) (*latest.ImageConfig, error) {
	var (
		dockerUsername = ""
		registryURL    = ""
		useKaniko      = false
		retImageConfig = &latest.ImageConfig{}
	)

	// Get docker client
	client, err := docker.NewClient(true)
	if err != nil {
		return nil, fmt.Errorf("Cannot create docker client: %v", err)
	}

	// Check if docker is installed
	for {
		_, err = client.Ping(contextpkg.Background())
		if err != nil {
			// Check if docker cli is installed
			err := exec.Command("docker").Run()
			if err == nil {
				useKaniko = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
					Question:               "Docker seems to be installed but is not running: " + err.Error() + " \nShould we build with kaniko instead?",
					DefaultValue:           "no",
					ValidationRegexPattern: "^(yes)|(no)$",
				}) == "yes"

				if useKaniko == false {
					continue
				}
			}

			// We use kaniko
			useKaniko = true

			// Set default build engine to kaniko, if no docker is installed
			retImageConfig.Build = &latest.BuildConfig{
				Kaniko: &latest.KanikoConfig{
					Cache: ptr.Bool(true),
				},
			}
		}

		break
	}

	// If not kaniko get docker hub credentials
	if cloudProvider == nil && useKaniko == false {
		log.StartWait("Checking Docker credentials")
		dockerAuthConfig, err := docker.GetAuthConfig(client, "", true)
		log.StopWait()

		if err == nil {
			dockerUsername = dockerAuthConfig.Username
		}

		// Don't push image in minikube
		if minikube.IsMinikube() {
			retImageConfig.Image = ptr.String("devspace")
			retImageConfig.SkipPush = ptr.Bool(true)
			return retImageConfig, nil
		}
	}

	// Check which registry to use
	if cloudProvider == nil {
		registryURL = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which registry do you want to push to? ('hub.docker.com' or URL)",
			DefaultValue:           "hub.docker.com",
			ValidationRegexPattern: "^.*$",
		})
	} else {
		// Get default registry
		provider, err := cloud.GetProvider(cloudProvider, log.GetInstance())
		if err != nil {
			return nil, fmt.Errorf("Error login into cloud provider: %v", err)
		}

		registries, err := provider.GetRegistries()
		if err != nil {
			return nil, fmt.Errorf("Error retrieving registries: %v", err)
		}
		if len(registries) > 0 {
			registryURL = registries[0].URL
		} else {
			registryURL = "hub.docker.com"
		}
	}

	// Determine username
	if registryURL != "hub.docker.com" {
		log.StartWait("Checking Docker credentials")
		dockerAuthConfig, err := docker.GetAuthConfig(client, registryURL, true)
		log.StopWait()
		if err != nil {
			return nil, fmt.Errorf("Couldn't find credentials in credentials store. Make sure you login to the registry with: docker login %s", registryURL)
		}

		dockerUsername = dockerAuthConfig.Username
	} else if dockerUsername == "" {
		log.Warn("No dockerhub credentials were found in the credentials store")
		log.Warn("Please make sure you have a https://hub.docker.com account")
		log.Warn("Installing docker is NOT required\n")

		for {
			dockerUsername = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "What is your docker hub username?",
				DefaultValue:           "",
				ValidationRegexPattern: "^.*$",
			})

			dockerPassword := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "What is your docker hub password?",
				DefaultValue:           "",
				ValidationRegexPattern: "^.*$",
				IsPassword:             true,
			})

			_, err = docker.Login(client, registryURL, dockerUsername, dockerPassword, false, true, true)
			if err != nil {
				log.Warn(err)
				continue
			}

			break
		}
	}

	// Image name to use
	defaultImageName := ""

	// Is docker hub?
	if registryURL == "hub.docker.com" {
		defaultImageName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which image name do you want to use on Docker Hub?",
			DefaultValue:           dockerUsername + "/devspace",
			ValidationRegexPattern: "^[a-zA-Z0-9/-]{4,60}$",
		})
	} else if regexp.MustCompile("^(.+\\.)?gcr.io$").Match([]byte(registryURL)) { // Is google registry?
		project, err := exec.Command("gcloud", "config", "get-value", "project").Output()
		gcloudProject := "myGCloudProject"

		if err == nil {
			gcloudProject = strings.TrimSpace(string(project))
		}

		defaultImageName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which image name do you want to push to?",
			DefaultValue:           registryURL + "/" + gcloudProject + "/devspace",
			ValidationRegexPattern: "^.*$",
		})
	} else if cloudProvider != nil {
		// Is DevSpace Cloud?
		defaultImageName = registryURL + "/${DEVSPACE_USERNAME}/" + DefaultImageName
	} else {
		defaultImageName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which image name do you want to push to?",
			DefaultValue:           registryURL + "/" + dockerUsername + "/devspace",
			ValidationRegexPattern: "^[a-zA-Z0-9\\./-]{4,90}$",
		})
	}

	// Check if we should create pull secrets for the image
	createPullSecret := true
	if cloudProvider == nil {
		createPullSecret = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question: "Do you want to enable automatic creation of pull secrets for this image?",
			Options:  []string{"yes", "no"},
		}) == "yes"
	}

	// Set image name
	retImageConfig.Image = &defaultImageName

	// Set image specifics
	if dockerfile != "" && dockerfile != image.DefaultDockerfilePath {
		if retImageConfig.Build == nil {
			retImageConfig.Build = &latest.BuildConfig{}
		}

		retImageConfig.Build.Dockerfile = &dockerfile
	}
	if context != "" && context != image.DefaultContextPath {
		if retImageConfig.Build == nil {
			retImageConfig.Build = &latest.BuildConfig{}
		}

		retImageConfig.Build.Context = &context
	}
	if createPullSecret {
		retImageConfig.CreatePullSecret = &createPullSecret
	}

	return retImageConfig, nil
}

//AddImage adds an image to the devspace
func AddImage(nameInConfig, name, tag, contextPath, dockerfilePath, buildEngine string) error {
	config := configutil.GetBaseConfig()

	imageConfig := &v1.ImageConfig{
		Image: &name,
		Build: &v1.BuildConfig{},
	}

	if tag != "" {
		imageConfig.Tag = &tag
	}
	if contextPath != "" {
		imageConfig.Build.Context = &contextPath
	}
	if dockerfilePath != "" {
		imageConfig.Build.Dockerfile = &dockerfilePath
	}

	if buildEngine == "docker" {
		imageConfig.Build.Docker = &v1.DockerConfig{}
	} else if buildEngine == "kaniko" {
		imageConfig.Build.Kaniko = &v1.KanikoConfig{}
	} else if buildEngine != "" {
		log.Errorf("BuildEngine %v unknown. Please select one of docker|kaniko", buildEngine)
	}

	if config.Images == nil {
		images := make(map[string]*v1.ImageConfig)
		config.Images = &images
	}

	(*config.Images)[nameInConfig] = imageConfig

	err := configutil.SaveLoadedConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %s", err.Error())
	}

	return nil
}

//RemoveImage removes an image from the devspace
func RemoveImage(removeAll bool, names []string) error {
	config := configutil.GetBaseConfig()

	if len(names) == 0 && removeAll == false {
		return fmt.Errorf("You have to specify at least one image")
	}

	newImageList := make(map[string]*v1.ImageConfig)

	if !removeAll && config.Images != nil {

	ImagesLoop:
		for nameInConfig, imageConfig := range *config.Images {
			for _, deletionName := range names {
				if deletionName == nameInConfig {
					continue ImagesLoop
				}
			}

			newImageList[nameInConfig] = imageConfig
		}
	}

	config.Images = &newImageList

	err := configutil.SaveLoadedConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %v", err)
	}

	return nil
}
