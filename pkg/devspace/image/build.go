package image

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/builder/kaniko"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/devspace/registry"
	"github.com/covexo/devspace/pkg/util/hash"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/randutil"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
)

// BuildAll builds all images
func BuildAll(client *kubernetes.Clientset, generatedConfig *generated.Config, isDev, forceRebuild bool, log log.Logger) (bool, error) {
	config := configutil.GetConfig()
	re := false

	for imageName, imageConf := range *config.Images {
		if imageConf.Build != nil && imageConf.Build.Disabled != nil && *imageConf.Build.Disabled == true {
			log.Infof("Skipping building image %s", imageName)
			continue
		}

		shouldRebuild, err := Build(client, generatedConfig, imageName, imageConf, isDev, forceRebuild, log)
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
func Build(client *kubernetes.Clientset, generatedConfig *generated.Config, imageConfigName string, imageConf *v1.ImageConfig, isDev, forceRebuild bool, log log.Logger) (bool, error) {
	dockerfilePath := "./Dockerfile"
	contextPath := "./"
	imageName := *imageConf.Name
	engineName := ""

	if imageConf.Build != nil {
		if imageConf.Build.DockerfilePath != nil {
			dockerfilePath = *imageConf.Build.DockerfilePath
		}

		if imageConf.Build.ContextPath != nil {
			contextPath = *imageConf.Build.ContextPath
		}
	}

	// Check if rebuild is needed
	needRebuild, err := shouldRebuild(generatedConfig, imageConf, contextPath, dockerfilePath, forceRebuild)
	if err != nil {
		return false, fmt.Errorf("Error during shouldRebuild check: %v", err)
	}
	if needRebuild == false {
		log.Infof("Skip building image '%s'", imageConfigName)
		return false, nil
	}

	// Get absolute paths
	absoluteDockerfilePath, err := filepath.Abs(dockerfilePath)
	if err != nil {
		return false, fmt.Errorf("Couldn't determine absolute path for %s", *imageConf.Build.DockerfilePath)
	}

	contextPath, err = filepath.Abs(contextPath)
	if err != nil {
		return false, fmt.Errorf("Couldn't determine absolute path for %s", *imageConf.Build.ContextPath)
	}

	// Get image tag
	imageTag, err := randutil.GenerateRandomString(7)
	if err != nil {
		return false, fmt.Errorf("Image building failed: %v", err)
	}
	if imageConf.Tag != nil {
		imageTag = *imageConf.Tag
	}

	// Create builder
	imageBuilder, err := CreateBuilder(client, generatedConfig, imageConf, imageTag)
	if err != nil {
		return false, err
	}

	if _, ok := imageBuilder.(*kaniko.Builder); ok {
		engineName = "kaniko"
	} else {
		engineName = "docker"
	}

	log.Infof("Building image '%s' with engine '%s'", imageName, engineName)

	// Display nice registry name
	displayRegistryURL := "hub.docker.com"
	registryURL, err := registry.GetRegistryFromImageName(imageName)
	if err != nil {
		return false, err
	}
	if registryURL != "" {
		displayRegistryURL = registryURL
	}

	// Authenticate
	if imageConf.SkipPush == nil || *imageConf.SkipPush == false {
		log.StartWait("Authenticating (" + displayRegistryURL + ")")
		_, err = imageBuilder.Authenticate()
		log.StopWait()

		if err != nil {
			return false, fmt.Errorf("Error during image registry authentication: %v", err)
		}

		log.Done("Authentication successful (" + displayRegistryURL + ")")
	}

	// Buildoptions
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

	// Check if we should overwrite entrypoint
	var entrypoint *[]*string
	if isDev {
		config := configutil.GetConfig()

		if config.Dev != nil && config.Dev.OverrideImages != nil {
			for _, imageOverrideConfig := range *config.Dev.OverrideImages {
				if *imageOverrideConfig.Name == imageConfigName {
					entrypoint = imageOverrideConfig.Entrypoint
					break
				}
			}
		}
	}

	// Build Image
	err = imageBuilder.BuildImage(contextPath, absoluteDockerfilePath, buildOptions, entrypoint)
	if err != nil {
		return false, fmt.Errorf("Error during image build: %v", err)
	}

	// Check if we skip push
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
	generatedConfig.GetActive().ImageTags[imageName] = imageTag

	log.Done("Done building and pushing image '" + imageName + "'")
	return true, nil
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
	excludes = append(excludes, ".devspace/")

	hash, err := hash.DirectoryExcludes(contextDir, excludes)
	if err != nil {
		return false, fmt.Errorf("Error hashing %s: %v", contextDir, err)
	}

	// When user has not used -b or --build flags
	activeConfig := runtimeConfig.GetActive()

	if forceRebuild == false {
		// only rebuild Docker image when Dockerfile or context has changed since latest build
		mustRebuild = activeConfig.DockerfileTimestamps[dockerfilePath] != dockerfileInfo.ModTime().Unix() || activeConfig.DockerContextPaths[contextPath] != hash
	}

	activeConfig.DockerfileTimestamps[dockerfilePath] = dockerfileInfo.ModTime().Unix()
	activeConfig.DockerContextPaths[contextPath] = hash

	// Check if there is an image tag for this image
	if _, ok := activeConfig.ImageTags[*imageConf.Name]; ok == false {
		return true, nil
	}

	return mustRebuild, nil
}
