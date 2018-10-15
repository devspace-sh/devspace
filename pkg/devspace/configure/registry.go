package configure

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/builder/docker"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/randutil"
	"github.com/covexo/devspace/pkg/util/stdinutil"
)

// ImageName configures the image name
func ImageName(dockerUsername string) error {
	config := configutil.GetConfig()
	registryURL := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Which registry do you want to push to? ('hub.docker.com' or URL)",
		DefaultValue:           "hub.docker.com",
		ValidationRegexPattern: "^.*$",
	})

	config.Services.InternalRegistry = nil

	if *registryURL != "hub.docker.com" {
		imageBuilder, err := docker.NewBuilder(*registryURL, "", "", false)
		if err == nil {
			log.StartWait("Checking Docker credentials")
			dockerAuthConfig, err := imageBuilder.Authenticate("", "", true)
			log.StopWait()

			if err != nil {
				return fmt.Errorf("Couldn't find credentials in credentials store. Make sure you login to the registry with: docker login %s", *registryURL)
			}

			dockerUsername = dockerAuthConfig.Username
		}
	} else if dockerUsername == "" {
		return fmt.Errorf("Make sure you login to docker hub with: docker login")
	}

	googleRegistryRegex := regexp.MustCompile("^(.+\\.)?gcr.io$")
	isGoogleRegistry := googleRegistryRegex.Match([]byte(*registryURL))
	isDockerHub := *registryURL == "hub.docker.com"
	defaultImageName := ""

	if isDockerHub {
		defaultImageName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which image name do you want to use on Docker Hub?",
			DefaultValue:           dockerUsername + "/devspace",
			ValidationRegexPattern: "^[a-zA-Z0-9/]{4,30}$",
		})
	} else if isGoogleRegistry {
		project, err := exec.Command("gcloud", "config", "get-value", "project").Output()
		gcloudProject := "myGCloudProject"

		if err == nil {
			gcloudProject = strings.TrimSpace(string(project))
		}

		defaultImageName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which image name do you want to push to?",
			DefaultValue:           *registryURL + "/" + gcloudProject + "/devspace",
			ValidationRegexPattern: "^.*$",
		})
	} else {
		defaultImageName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which image name do you want to push to?",
			DefaultValue:           *registryURL + "/" + dockerUsername + "/devspace",
			ValidationRegexPattern: "^[a-zA-Z0-9\\./]{4,30}$",
		})
	}

	imageMap := *config.Images
	imageMap["default"].Name = &defaultImageName

	return nil
}

// InternalRegistry configures the internal registry
func InternalRegistry() error {
	config := configutil.GetConfig()
	overwriteConfig := configutil.GetOverwriteConfig()
	internalRegistryConfig := config.Services.InternalRegistry

	imageMap := *config.Images
	defaultImageConf, defaultImageExists := imageMap["default"]

	if defaultImageExists {
		defaultImageConf.Registry = configutil.String("internal")
	}

	if internalRegistryConfig == nil {
		internalRegistryConfig = &v1.InternalRegistry{
			Release: &v1.Release{},
		}
		config.Services.InternalRegistry = internalRegistryConfig
	}

	if internalRegistryConfig.Release.Name == nil {
		internalRegistryConfig.Release.Name = configutil.String("devspace-registry")
	}
	if internalRegistryConfig.Release.Namespace == nil {
		internalRegistryConfig.Release.Namespace = config.DevSpace.Release.Namespace
	}

	overwriteRegistryMap := *overwriteConfig.Registries
	overwriteRegistryConfig, overwriteRegistryConfigFound := overwriteRegistryMap["internal"]

	if !overwriteRegistryConfigFound {
		overwriteRegistryConfig = &v1.RegistryConfig{
			Auth: &v1.RegistryAuth{},
		}
		overwriteRegistryMap["internal"] = overwriteRegistryConfig
	}

	registryAuth := overwriteRegistryConfig.Auth
	if registryAuth.Username == nil {
		randomUserSuffix, err := randutil.GenerateRandomString(5)
		if err != nil {
			return fmt.Errorf("Error creating random username: %s", err.Error())
		}

		registryAuth.Username = configutil.String("user-" + randomUserSuffix)
	}

	if registryAuth.Password == nil {
		randomPassword, err := randutil.GenerateRandomString(12)
		if err != nil {
			return fmt.Errorf("Error creating random password: %s", err.Error())
		}

		registryAuth.Password = &randomPassword
	}

	var registryReleaseValues map[interface{}]interface{}
	if internalRegistryConfig.Release.Values != nil {
		registryReleaseValues = *internalRegistryConfig.Release.Values
	} else {
		registryReleaseValues = map[interface{}]interface{}{}

		registryDomain := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which domain should your container registry be using? (optional, requires an ingress controller)",
			ValidationRegexPattern: "^(([a-z0-9]([a-z0-9-]{0,120}[a-z0-9])?\\.)+[a-z0-9]{2,})?$",
		})

		if *registryDomain != "" {
			registryReleaseValues = map[interface{}]interface{}{
				"Ingress": map[string]interface{}{
					"Enabled": true,
					"Hosts": []string{
						*registryDomain,
					},
					"Annotations": map[string]string{
						"Kubernetes.io/tls-acme": "true",
					},
					"Tls": []map[string]interface{}{
						map[string]interface{}{
							"SecretName": "tls-devspace-registry",
							"Hosts": []string{
								*registryDomain,
							},
						},
					},
				},
			}
		} else if kubectl.IsMinikube() == false {
			log.Warn("Your Kubernetes cluster will not be able to pull images from a registry without a registry domain!\n")
		}
	}

	secrets, registryHasSecrets := registryReleaseValues["secrets"]
	if !registryHasSecrets {
		secrets = map[interface{}]interface{}{}
		registryReleaseValues["secrets"] = secrets
	}

	secretMap, secretsIsMap := secrets.(map[interface{}]interface{})
	if secretsIsMap {
		_, registryHasSecretHtpasswd := secretMap["htpasswd"]
		if !registryHasSecretHtpasswd {
			secretMap["htpasswd"] = ""
		}
	}

	internalRegistryConfig.Release.Values = &registryReleaseValues
	config.Registries = &overwriteRegistryMap

	return nil
}
