package image

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/pkg/devspace/builder/kaniko"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
)

type builderConfig struct {
	client kubernetes.Interface

	imageConfigName string
	imageConf       *v1.ImageConfig

	dockerfilePath string
	contextPath    string

	imageName  string
	engineName string

	entrypoint *[]*string
}

func newBuilderConfig(client kubernetes.Interface, imageConfigName string, imageConf *v1.ImageConfig, isDev bool) *builderConfig {
	var (
		dockerfilePath, contextPath = getDockerfileAndContext(imageConfigName, imageConf, isDev)
		imageName, engineName       = *imageConf.Image, ""
	)

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

	return &builderConfig{
		client: client,

		imageConfigName: imageConfigName,
		imageConf:       imageConf,

		dockerfilePath: dockerfilePath,
		contextPath:    contextPath,

		imageName:  imageName,
		engineName: engineName,

		entrypoint: entrypoint,
	}
}

// Build builds an image with the specified engine and returns the image tag
func (b *builderConfig) Build(imageTag string, log log.Logger) error {
	// Get absolute paths
	absoluteDockerfilePath, err := filepath.Abs(b.dockerfilePath)
	if err != nil {
		return fmt.Errorf("Couldn't determine absolute path for %s", *b.imageConf.Build.Dockerfile)
	}

	absoluteContextPath, err := filepath.Abs(b.contextPath)
	if err != nil {
		return fmt.Errorf("Couldn't determine absolute path for %s", *b.imageConf.Build.Context)
	}

	// Create actual builder from config
	imageBuilder, err := CreateBuilder(b.client, b.imageConf, imageTag, log)
	if err != nil {
		return err
	}

	if _, ok := imageBuilder.(*kaniko.Builder); ok {
		b.engineName = "kaniko"
	} else {
		b.engineName = "docker"
	}

	log.Infof("Building image '%s' with engine '%s'", b.imageName, b.engineName)

	// Display nice registry name
	displayRegistryURL := "hub.docker.com"
	registryURL, err := registry.GetRegistryFromImageName(b.imageName)
	if err != nil {
		return err
	}
	if registryURL != "" {
		displayRegistryURL = registryURL
	}

	// Authenticate
	if b.imageConf.SkipPush == nil || *b.imageConf.SkipPush == false {
		log.StartWait("Authenticating (" + displayRegistryURL + ")")
		_, err = imageBuilder.Authenticate()
		log.StopWait()
		if err != nil {
			return fmt.Errorf("Error during image registry authentication: %v", err)
		}

		log.Done("Authentication successful (" + displayRegistryURL + ")")
	}

	// Buildoptions
	buildOptions := &types.ImageBuildOptions{}
	if b.imageConf.Build != nil && b.imageConf.Build.Options != nil {
		if b.imageConf.Build.Options.BuildArgs != nil {
			buildOptions.BuildArgs = *b.imageConf.Build.Options.BuildArgs
		}
		if b.imageConf.Build.Options.Target != nil {
			buildOptions.Target = *b.imageConf.Build.Options.Target
		}
		if b.imageConf.Build.Options.Network != nil {
			buildOptions.NetworkMode = *b.imageConf.Build.Options.Network
		}
	}

	// Build Image
	err = imageBuilder.BuildImage(absoluteContextPath, absoluteDockerfilePath, buildOptions, b.entrypoint)
	if err != nil {
		return fmt.Errorf("Error during image build: %v", err)
	}

	// Check if we skip push
	if b.imageConf.SkipPush == nil || *b.imageConf.SkipPush == false {
		err = imageBuilder.PushImage()
		if err != nil {
			return fmt.Errorf("Error during image push: %v", err)
		}

		log.Info("Image pushed to registry (" + displayRegistryURL + ")")
	} else {
		log.Infof("Skip image push for %s", b.imageName)
	}

	log.Done("Done processing image '" + b.imageName + "'")
	return nil
}

func (b *builderConfig) shouldRebuild(cache *generated.CacheConfig) (bool, error) {
	// Hash dockerfile
	_, err := os.Stat(b.dockerfilePath)
	if err != nil {
		return false, fmt.Errorf("Dockerfile %s missing: %v", b.dockerfilePath, err)
	}
	dockerfileHash, err := hash.Directory(b.dockerfilePath)
	if err != nil {
		return false, errors.Wrap(err, "hash dockerfile")
	}

	// Hash context path
	contextDir, relDockerfile, err := build.GetContextFromLocalDir(b.contextPath, b.dockerfilePath)
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

	contextHash, err := hash.DirectoryExcludes(contextDir, excludes)
	if err != nil {
		return false, fmt.Errorf("Error hashing %s: %v", contextDir, err)
	}

	imageCache := cache.GetImageCache(b.imageConfigName)

	// Hash image config
	configStr, err := yaml.Marshal(*b.imageConf)
	if err != nil {
		return false, errors.Wrap(err, "marshal image config")
	}

	imageConfigHash := hash.String(string(configStr))

	// Hash entrypoint
	entrypointHash := ""
	if b.entrypoint != nil {
		for _, str := range *b.entrypoint {
			entrypointHash += *str
		}

		entrypointHash = hash.String(string(entrypointHash))
	}

	// only rebuild Docker image when Dockerfile or context has changed since latest build
	mustRebuild := imageCache.Tag == "" || imageCache.DockerfileHash != dockerfileHash || imageCache.ContextHash != contextHash || imageCache.ImageConfigHash != imageConfigHash || imageCache.EntrypointHash != entrypointHash

	imageCache.DockerfileHash = dockerfileHash
	imageCache.ContextHash = contextHash
	imageCache.ImageConfigHash = imageConfigHash
	imageCache.EntrypointHash = entrypointHash

	return mustRebuild, nil
}

func getDockerfileAndContext(imageConfigName string, imageConf *v1.ImageConfig, isDev bool) (string, string) {
	var (
		config         = configutil.GetConfig()
		dockerfilePath = DefaultDockerfilePath
		contextPath    = DefaultContextPath
	)

	if imageConf.Build != nil {
		if imageConf.Build.Dockerfile != nil {
			dockerfilePath = *imageConf.Build.Dockerfile
		}

		if imageConf.Build.Context != nil {
			contextPath = *imageConf.Build.Context
		}
	}

	if isDev && config.Dev != nil && config.Dev.OverrideImages != nil {
		for _, overrideConfig := range *config.Dev.OverrideImages {
			if *overrideConfig.Name == imageConfigName {
				if overrideConfig.Dockerfile != nil {
					dockerfilePath = *overrideConfig.Dockerfile
				}
				if overrideConfig.Context != nil {
					contextPath = *overrideConfig.Context
				}
			}
		}
	}

	return dockerfilePath, contextPath
}
