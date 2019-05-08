package helper

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// BuildHelper is the helper class to store common functionality used by both the docker and kaniko builder
type BuildHelper struct {
	ImageConfigName string
	ImageConf       *latest.ImageConfig
	Config          *latest.Config

	DockerfilePath string
	ContextPath    string

	EngineName string
	ImageName  string
	ImageTag   string
	Entrypoint *[]*string
}

// BuildHelperInterface is the interface the build helper uses to build an image
type BuildHelperInterface interface {
	BuildImage(absoluteContextPath string, absoluteDockerfilePath string, entrypoint *[]*string, log log.Logger) error
}

// NewBuildHelper creates a new build helper for a certain engine
func NewBuildHelper(config *latest.Config, engineName string, imageConfigName string, imageConf *latest.ImageConfig, imageTag string, isDev bool) *BuildHelper {
	var (
		dockerfilePath, contextPath = GetDockerfileAndContext(config, imageConfigName, imageConf, isDev)
		imageName                   = *imageConf.Image
	)

	// Check if we should overwrite entrypoint
	var entrypoint *[]*string
	if isDev {
		if config.Dev != nil && config.Dev.OverrideImages != nil {
			for _, imageOverrideConfig := range *config.Dev.OverrideImages {
				if *imageOverrideConfig.Name == imageConfigName {
					entrypoint = imageOverrideConfig.Entrypoint
					break
				}
			}
		}
	}

	return &BuildHelper{
		ImageConfigName: imageConfigName,
		ImageConf:       imageConf,

		DockerfilePath: dockerfilePath,
		ContextPath:    contextPath,

		ImageName:  imageName,
		ImageTag:   imageTag,
		EngineName: engineName,

		Entrypoint: entrypoint,
		Config:     config,
	}
}

// Build builds a new image
func (b *BuildHelper) Build(imageBuilder BuildHelperInterface, log log.Logger) error {
	// Get absolute paths
	absoluteDockerfilePath, err := filepath.Abs(b.DockerfilePath)
	if err != nil {
		return fmt.Errorf("Couldn't determine absolute path for %s", b.DockerfilePath)
	}

	absoluteContextPath, err := filepath.Abs(b.ContextPath)
	if err != nil {
		return fmt.Errorf("Couldn't determine absolute path for %s", b.ContextPath)
	}

	log.Infof("Building image '%s' with engine '%s'", b.ImageName, b.EngineName)

	// Build Image
	err = imageBuilder.BuildImage(absoluteContextPath, absoluteDockerfilePath, b.Entrypoint, log)
	if err != nil {
		return fmt.Errorf("Error during image build: %v", err)
	}

	log.Done("Done processing image '" + b.ImageName + "'")
	return nil
}

// ShouldRebuild determines if the image should be rebuilt
func (b *BuildHelper) ShouldRebuild(cache *generated.CacheConfig) (bool, error) {
	// Hash dockerfile
	_, err := os.Stat(b.DockerfilePath)
	if err != nil {
		return false, fmt.Errorf("Dockerfile %s missing: %v", b.DockerfilePath, err)
	}
	dockerfileHash, err := hash.Directory(b.DockerfilePath)
	if err != nil {
		return false, errors.Wrap(err, "hash dockerfile")
	}

	// Hash context path
	contextDir, relDockerfile, err := build.GetContextFromLocalDir(b.ContextPath, b.DockerfilePath)
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

	imageCache := cache.GetImageCache(b.ImageConfigName)

	// Hash image config
	configStr, err := yaml.Marshal(*b.ImageConf)
	if err != nil {
		return false, errors.Wrap(err, "marshal image config")
	}

	imageConfigHash := hash.String(string(configStr))

	// Hash entrypoint
	entrypointHash := ""
	if b.Entrypoint != nil {
		for _, str := range *b.Entrypoint {
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
