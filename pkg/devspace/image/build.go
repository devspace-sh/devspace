package image

import (
	"errors"
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
	dockerclient "github.com/covexo/devspace/pkg/devspace/docker"
	"github.com/covexo/devspace/pkg/devspace/registry"
	"github.com/covexo/devspace/pkg/util/hash"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/randutil"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
)

// BuildAll builds all images
func BuildAll(client *kubernetes.Clientset, generatedConfig *generated.Config, forceRebuild bool, log log.Logger) (bool, error) {
	config := configutil.GetConfig()
	re := false

	for imageName, imageConf := range *config.Images {
		if imageConf.Build != nil && imageConf.Build.Disabled != nil && *imageConf.Build.Disabled == true {
			log.Infof("Skipping building image %s", imageName)
			continue
		}

		shouldRebuild, err := Build(client, generatedConfig, imageName, imageConf, forceRebuild, log)
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
func Build(client *kubernetes.Clientset, generatedConfig *generated.Config, imageName string, imageConf *v1.ImageConfig, forceRebuild bool, log log.Logger) (bool, error) {
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

	if needRebuild, err := shouldRebuild(generatedConfig, imageConf, contextPath, dockerfilePath, forceRebuild); needRebuild || err != nil {
		if err != nil {
			return false, fmt.Errorf("Error during shouldRebuild check: %v", err)
		}

		absoluteDockerfilePath, err := filepath.Abs(dockerfilePath)
		if err != nil {
			return false, fmt.Errorf("Couldn't determine absolute path for %s", *imageConf.Build.DockerfilePath)
		}

		contextPath, err = filepath.Abs(contextPath)
		if err != nil {
			return false, fmt.Errorf("Couldn't determine absolute path for %s", *imageConf.Build.ContextPath)
		}

		var imageBuilder builder.Interface
		rebuild = true

		imageTag, err := randutil.GenerateRandomString(7)
		if err != nil {
			return false, fmt.Errorf("Image building failed: %v", err)
		}
		if imageConf.Tag != nil {
			imageTag = *imageConf.Tag
		}

		imageName, registryConf, err := registry.GetRegistryConfigFromImageConfig(imageConf)
		if err != nil {
			return false, fmt.Errorf("GetRegistryConfigFromImageConfig failed: %v", err)
		}

		engineName := ""

		if imageConf.Build != nil && imageConf.Build.Kaniko != nil {
			engineName = "kaniko"
			buildNamespace, err := configutil.GetDefaultNamespace(config)
			if err != nil {
				return false, errors.New("Error retrieving default namespace")
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

			dockerClient, err := dockerclient.NewClient(false)
			if err != nil {
				return false, fmt.Errorf("Error creating docker client: %v", err)
			}

			imageBuilder, err = kaniko.NewBuilder(*registryConf.URL, pullSecret, imageName, imageTag, (*generatedConfig).ImageTags[imageName], buildNamespace, dockerClient, client, allowInsecurePush)
			if err != nil {
				return false, fmt.Errorf("Error creating kaniko builder: %v", err)
			}
		} else {
			engineName = "docker"

			preferMinikube := true
			if imageConf.Build != nil && imageConf.Build.Docker != nil && imageConf.Build.Docker.PreferMinikube != nil {
				preferMinikube = *imageConf.Build.Docker.PreferMinikube
			}

			dockerClient, err := dockerclient.NewClient(preferMinikube)
			if err != nil {
				return false, fmt.Errorf("Error creating docker client: %v", err)
			}

			imageBuilder, err = docker.NewBuilder(dockerClient, *registryConf.URL, imageName, imageTag)
			if err != nil {
				return false, fmt.Errorf("Error creating docker builder: %v", err)
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
		if *registryConf.URL != "" {
			displayRegistryURL = *registryConf.URL
		}

		log.StartWait("Authenticating (" + displayRegistryURL + ")")
		_, err = imageBuilder.Authenticate(username, password, len(username) == 0)
		log.StopWait()

		if err != nil {
			return false, fmt.Errorf("Error during image registry authentication: %v", err)
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

		err = imageBuilder.BuildImage(contextPath, absoluteDockerfilePath, buildOptions)
		if err != nil {
			return false, fmt.Errorf("Error during image build: %v", err)
		}

		if imageConf.SkipPush == nil || *imageConf.SkipPush == false {
			err = imageBuilder.PushImage()
			if err != nil {
				return false, fmt.Errorf("Error during image push: %v", err)
			}

			log.Info("Image pushed to registry (" + displayRegistryURL + ")")
		} else {
			log.Infof("Skip image push for %s", imageName)
		}

		// Update config
		if *registryConf.URL != "" {
			imageName = *registryConf.URL + "/" + imageName
		}

		generatedConfig.ImageTags[imageName] = imageTag

		log.Done("Done building and pushing image '" + imageName + "'")

	} else {
		log.Infof("Skip building image '%s'", imageName)
	}

	return rebuild, nil
}

func shouldRebuild(runtimeConfig *generated.Config, imageConf *v1.ImageConfig, contextPath, dockerfilePath string, forceRebuild bool) (bool, error) {
	mustRebuild := true

	// Get dockerfile timestamp
	dockerfileInfo, err := os.Stat(dockerfilePath)
	if err != nil {
		return false, fmt.Errorf("Dockerfile %s missing: %v", dockerfilePath, err)
	}

	// Hash context path
	contextDir, relDockerfile, err := build.GetContextFromLocalDir(contextPath, dockerfilePath)
	if err != nil {
		return false, err
	}

	excludes, err := build.ReadDockerignore(contextDir)
	if err != nil {
		return false, fmt.Errorf("Error reading .dockerignore: %v", err)
	}

	relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)
	excludes = build.TrimBuildFilesFromExcludes(excludes, relDockerfile, false)

	hash, err := hash.DirectoryExcludes(contextDir, excludes)
	if err != nil {
		return false, err
	}

	// When user has not used -b or --build flags
	if forceRebuild == false {
		// only rebuild Docker image when Dockerfile or context has changed since latest build
		mustRebuild = runtimeConfig.DockerfileTimestamps[dockerfilePath] != dockerfileInfo.ModTime().Unix() || runtimeConfig.DockerContextPaths[contextPath] != hash
	}

	runtimeConfig.DockerfileTimestamps[dockerfilePath] = dockerfileInfo.ModTime().Unix()
	runtimeConfig.DockerContextPaths[contextPath] = hash

	return mustRebuild, nil
}
