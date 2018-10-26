package image

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/builder"
	"github.com/covexo/devspace/pkg/devspace/builder/docker"
	"github.com/covexo/devspace/pkg/devspace/builder/kaniko"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/registry"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/randutil"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	dockerregistry "github.com/docker/docker/registry"
)

// BuildAll builds all images
func BuildAll(client *kubernetes.Clientset, generatedConfig *generated.Config, forceRebuild bool) (bool, error) {
	config := configutil.GetConfig()
	re := false

	for imageName, imageConf := range *config.Images {
		shouldRebuild, err := Build(client, generatedConfig, imageName, imageConf, forceRebuild)
		if err != nil {
			return false, err
		}

		if shouldRebuild {
			re = true
		}
	}

	return re, nil
}

// Build builds an image with the specified engine
func Build(client *kubernetes.Clientset, generatedConfig *generated.Config, imageName string, imageConf *v1.ImageConfig, forceRebuild bool) (bool, error) {
	rebuild := false
	config := configutil.GetConfig()
	dockerfilePath := "./Dockerfile"
	contextPath := "./"

	if imageConf.Build != nil {
		if imageConf.Build.DockerfilePath != nil {
			dockerfilePath = *imageConf.Build.DockerfilePath
		}

		if imageConf.Build.ContextPath != nil {
			contextPath = *imageConf.Build.ContextPath
		}
	}

	dockerfilePath, err := filepath.Abs(dockerfilePath)
	if err != nil {
		return false, fmt.Errorf("Couldn't determine absolute path for %s", *imageConf.Build.DockerfilePath)
	}

	contextPath, err = filepath.Abs(contextPath)
	if err != nil {
		return false, fmt.Errorf("Couldn't determine absolute path for %s", *imageConf.Build.ContextPath)
	}

	if shouldRebuild(generatedConfig, imageConf, dockerfilePath, forceRebuild) {
		rebuild = true
		imageTag, randErr := randutil.GenerateRandomString(7)
		if randErr != nil {
			return false, fmt.Errorf("Image building failed: %s", randErr.Error())
		}

		var registryConf *v1.RegistryConfig
		var imageBuilder builder.Interface

		engineName := ""
		registryURL := ""
		imageName := *imageConf.Name

		if imageConf.Registry != nil {
			registryConf, err = registry.GetRegistryConfig(imageConf)
			if err != nil {
				return false, err
			}

			if registryConf.URL != nil {
				registryURL = *registryConf.URL
			}
			if registryURL == "hub.docker.com" {
				registryURL = ""
			}
		} else {
			registryURL, err = GetRegistryFromImageName(*imageConf.Name)
			if err != nil {
				return false, err
			}

			if len(registryURL) > 0 {
				// Crop registry Url from imageName
				imageName = imageName[len(registryURL)+1:]
			}

			registryConf = &v1.RegistryConfig{
				URL:      &registryURL,
				Insecure: configutil.Bool(false),
			}
		}

		if imageConf.Build != nil && imageConf.Build.Kaniko != nil {
			engineName = "kaniko"
			buildNamespace, err := configutil.GetDefaultNamespace(config)
			if err != nil {
				log.Fatalf("Error retrieving default namespace")
			}

			if imageConf.Build.Kaniko.Namespace != nil && *imageConf.Build.Kaniko.Namespace != "" {
				buildNamespace = *imageConf.Build.Kaniko.Namespace
			}

			allowInsecurePush := false
			if registryConf.Insecure != nil {
				allowInsecurePush = *registryConf.Insecure
			}

			pullSecret := ""
			if imageConf.Build.Kaniko.PullSecret != nil {
				pullSecret = *imageConf.Build.Kaniko.PullSecret
			}

			imageBuilder, err = kaniko.NewBuilder(registryURL, pullSecret, imageName, imageTag, (*generatedConfig).ImageTags[imageName], buildNamespace, client, allowInsecurePush)
			if err != nil {
				log.Fatalf("Error creating kaniko builder: %v", err)
			}
		} else {
			engineName = "docker"

			preferMinikube := true
			if imageConf.Build != nil && imageConf.Build.Docker != nil && imageConf.Build.Docker.PreferMinikube != nil {
				preferMinikube = *imageConf.Build.Docker.PreferMinikube
			}

			imageBuilder, err = docker.NewBuilder(registryURL, imageName, imageTag, preferMinikube)
			if err != nil {
				log.Fatalf("Error creating docker client: %v", err)
			}
		}

		log.Infof("Building image '%s' with engine '%s'", imageName, engineName)

		username := ""
		password := ""
		if registryConf.Auth != nil {
			if registryConf.Auth.Username != nil {
				username = *registryConf.Auth.Username
			}

			if registryConf.Auth.Password != nil {
				password = *registryConf.Auth.Password
			}
		}

		displayRegistryURL := "hub.docker.com"
		if registryURL != "" {
			displayRegistryURL = registryURL
		}

		log.StartWait("Authenticating (" + displayRegistryURL + ")")
		_, err = imageBuilder.Authenticate(username, password, len(username) == 0)
		log.StopWait()

		if err != nil {
			log.Fatalf("Error during image registry authentication: %v", err)
		}

		log.Done("Authentication successful (" + displayRegistryURL + ")")

		buildOptions := &types.ImageBuildOptions{}

		if imageConf.Build != nil && imageConf.Build.Options != nil {
			if imageConf.Build.Options.BuildArgs != nil {
				buildOptions.BuildArgs = *imageConf.Build.Options.BuildArgs
			}
			if imageConf.Build.Options.Target != nil {
				buildOptions.Target = *imageConf.Build.Options.Target
			}
			if imageConf.Build.Options.Network != nil {
				buildOptions.NetworkMode = *imageConf.Build.Options.Network
			}
		}

		err = imageBuilder.BuildImage(contextPath, dockerfilePath, buildOptions)
		if err != nil {
			return false, fmt.Errorf("Error during image build: %v", err)
		}

		err = imageBuilder.PushImage()
		if err != nil {
			return false, fmt.Errorf("Error during image push: %v", err)
		}

		log.Info("Image pushed to registry (" + displayRegistryURL + ")")

		// Update config
		if registryURL != "" {
			imageName = registryURL + "/" + imageName
		}

		generatedConfig.ImageTags[imageName] = imageTag

		log.Done("Done building and pushing image '" + imageName + "'")
	} else {
		log.Infof("Skip building image '%s'", imageName)
	}

	return rebuild, nil
}

func shouldRebuild(runtimeConfig *generated.Config, imageConf *v1.ImageConfig, dockerfilePath string, forceRebuild bool) bool {
	mustRebuild := true
	dockerfileInfo, err := os.Stat(dockerfilePath)

	if err != nil {
		log.Warnf("Dockerfile %s missing: %v", dockerfilePath, err)
		mustRebuild = false
	} else {
		// When user has not used -b or --build flags
		if forceRebuild == false {
			// only rebuild Docker image when Dockerfile has changed since latest build
			mustRebuild = dockerfileInfo.ModTime().Unix() != runtimeConfig.DockerLatestTimestamps[dockerfilePath]
		}

		runtimeConfig.DockerLatestTimestamps[dockerfilePath] = dockerfileInfo.ModTime().Unix()
	}

	return mustRebuild
}

// GetRegistryFromImageName retrieves the registry name from an imageName
func GetRegistryFromImageName(imageName string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return "", err
	}

	repoInfo, err := dockerregistry.ParseRepositoryInfo(ref)
	if err != nil {
		return "", err
	}

	if repoInfo.Index.Official {
		return "", nil
	}

	return repoInfo.Index.Name, nil
}
