package configure

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/docker"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/stdinutil"
)

// Image configures the image name
func Image(dockerUsername string, skipQuestions bool, registryURL, defaultImageName string, createPullSecret bool) error {
	config := configutil.GetConfig()

	if skipQuestions {
		registryURL = "hub.docker.com"
	}
	if registryURL == "" {
		registryURL = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which registry do you want to push to? ('hub.docker.com' or URL)",
			DefaultValue:           "hub.docker.com",
			ValidationRegexPattern: "^.*$",
		})
	}

	client, err := docker.NewClient(false)
	if err != nil {
		return fmt.Errorf("Couldn't create docker client: %v", err)
	}

	if registryURL != "hub.docker.com" {
		log.StartWait("Checking Docker credentials")
		dockerAuthConfig, err := docker.GetAuthConfig(client, registryURL, true)
		log.StopWait()
		if err != nil {
			return fmt.Errorf("Couldn't find credentials in credentials store. Make sure you login to the registry with: docker login %s", registryURL)
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

			_, err = docker.Login(client, registryURL, dockerUsername, dockerPassword, true, true)
			if err != nil {
				log.Warn(err)
				continue
			}

			break
		}
	}

	googleRegistryRegex := regexp.MustCompile("^(.+\\.)?gcr.io$")
	isGoogleRegistry := googleRegistryRegex.Match([]byte(registryURL))
	isDockerHub := registryURL == "hub.docker.com"

	if skipQuestions {
		defaultImageName = dockerUsername + "/devspace"
	} else {
		if isDockerHub {
			defaultImageName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Which image name do you want to use on Docker Hub?",
				DefaultValue:           dockerUsername + "/devspace",
				ValidationRegexPattern: "^[a-zA-Z0-9/-]{4,60}$",
			})
		} else if isGoogleRegistry {
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
		} else {
			defaultImageName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Which image name do you want to push to?",
				DefaultValue:           registryURL + "/" + dockerUsername + "/devspace",
				ValidationRegexPattern: "^[a-zA-Z0-9\\./-]{4,90}$",
			})
		}

		createPullSecret = createPullSecret || *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Do you want to enable automatic creation of pull secrets for this image? (yes | no)",
			DefaultValue:           "no",
			ValidationRegexPattern: "^(yes|no)$",
		}) == "yes"
	}

	imageMap := *config.Images
	imageMap["default"].Name = &defaultImageName

	if createPullSecret {
		imageMap["default"].CreatePullSecret = &createPullSecret
	}

	return nil
}
