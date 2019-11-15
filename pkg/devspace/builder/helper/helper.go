package helper

import (
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
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
	Entrypoint []string
	Cmd        []string

	IsDev      bool
	KubeClient kubectl.Client
}

// BuildHelperInterface is the interface the build helper uses to build an image
type BuildHelperInterface interface {
	BuildImage(absoluteContextPath string, absoluteDockerfilePath string, entrypoint []string, cmd []string, log log.Logger) error
}

// NewBuildHelper creates a new build helper for a certain engine
func NewBuildHelper(config *latest.Config, kubeClient kubectl.Client, engineName string, imageConfigName string, imageConf *latest.ImageConfig, imageTag string, isDev bool) *BuildHelper {
	var (
		dockerfilePath, contextPath = GetDockerfileAndContext(config, imageConfigName, imageConf, isDev)
		imageName                   = imageConf.Image
	)

	// Check if we should overwrite entrypoint
	var (
		entrypoint []string
		cmd        []string
	)
	if isDev {
		if config.Dev != nil && config.Dev.Interactive != nil {
			for _, imageOverrideConfig := range config.Dev.Interactive.Images {
				if imageOverrideConfig.Name == imageConfigName {
					entrypoint = imageOverrideConfig.Entrypoint
					cmd = imageOverrideConfig.Cmd
					break
				}
			}
		}
	}

	if entrypoint == nil && imageConf.Entrypoint != nil {
		entrypoint = imageConf.Entrypoint
	}
	if cmd == nil && imageConf.Cmd != nil {
		cmd = imageConf.Cmd
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
		Cmd:        cmd,
		Config:     config,

		IsDev:      isDev,
		KubeClient: kubeClient,
	}
}

// Build builds a new image
func (b *BuildHelper) Build(imageBuilder BuildHelperInterface, log log.Logger) error {
	// Get absolute paths
	absoluteDockerfilePath, err := filepath.Abs(b.DockerfilePath)
	if err != nil {
		return errors.Errorf("Couldn't determine absolute path for %s", b.DockerfilePath)
	}

	absoluteContextPath, err := filepath.Abs(b.ContextPath)
	if err != nil {
		return errors.Errorf("Couldn't determine absolute path for %s", b.ContextPath)
	}

	log.Infof("Building image '%s:%s' with engine '%s'", b.ImageName, b.ImageTag, b.EngineName)

	// Build Image
	err = imageBuilder.BuildImage(absoluteContextPath, absoluteDockerfilePath, b.Entrypoint, b.Cmd, log)
	if err != nil {
		return errors.Errorf("Error during image build: %v", err)
	}

	log.Done("Done processing image '" + b.ImageName + "'")
	return nil
}

// ShouldRebuild determines if the image should be rebuilt
func (b *BuildHelper) ShouldRebuild(cache *generated.CacheConfig, ignoreContextPathChanges bool) (bool, error) {
	imageCache := cache.GetImageCache(b.ImageConfigName)

	// Hash dockerfile
	_, err := os.Stat(b.DockerfilePath)
	if err != nil {
		return false, errors.Errorf("Dockerfile %s missing: %v", b.DockerfilePath, err)
	}
	dockerfileHash, err := hash.Directory(b.DockerfilePath)
	if err != nil {
		return false, errors.Wrap(err, "hash dockerfile")
	}

	// Hash image config
	configStr, err := yaml.Marshal(*b.ImageConf)
	if err != nil {
		return false, errors.Wrap(err, "marshal image config")
	}

	imageConfigHash := hash.String(string(configStr))

	// Hash entrypoint
	entrypointHash := ""
	if len(b.Entrypoint) > 0 {
		for _, str := range b.Entrypoint {
			entrypointHash += str
		}
	}
	if len(b.Cmd) > 0 {
		for _, str := range b.Cmd {
			entrypointHash += str
		}
	}
	if entrypointHash != "" {
		entrypointHash = hash.String(entrypointHash)
	}

	// only rebuild Docker image when Dockerfile or context has changed since latest build
	mustRebuild := imageCache.Tag == "" || imageCache.DockerfileHash != dockerfileHash || imageCache.ImageConfigHash != imageConfigHash || imageCache.EntrypointHash != entrypointHash

	// Check if we really should skip context path changes, this is only the case if we find a sync config for the given image name
	if ignoreContextPathChanges {
		ignoreContextPathChanges = false
		if b.Config.Dev != nil && imageCache.ImageName != "" {
			for _, syncConfig := range b.Config.Dev.Sync {
				if syncConfig.ImageName == b.ImageConfigName {
					ignoreContextPathChanges = true
					break
				}
			}
		}
	}

	if ignoreContextPathChanges == false {
		// Hash context path
		contextDir, relDockerfile, err := build.GetContextFromLocalDir(b.ContextPath, b.DockerfilePath)
		if err != nil {
			return false, errors.Wrap(err, "get context from local dir")
		}

		excludes, err := build.ReadDockerignore(contextDir)
		if err != nil {
			return false, errors.Errorf("Error reading .dockerignore: %v", err)
		}

		relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)
		excludes = build.TrimBuildFilesFromExcludes(excludes, relDockerfile, false)
		excludes = append(excludes, ".devspace/")

		contextHash, err := hash.DirectoryExcludes(contextDir, excludes, false)
		if err != nil {
			return false, errors.Errorf("Error hashing %s: %v", contextDir, err)
		}

		mustRebuild = mustRebuild || imageCache.ContextHash != contextHash
		imageCache.ContextHash = contextHash
	}

	imageCache.DockerfileHash = dockerfileHash
	imageCache.ImageConfigHash = imageConfigHash
	imageCache.EntrypointHash = entrypointHash

	// Okay this check verifies if the previous deploy context was local kubernetes context where we didn't push the image and now have a kubernetes context where we probably push
	// or use another docker client (e.g. minikube <-> docker-desktop)
	if b.KubeClient != nil && cache.LastContext != nil && cache.LastContext.Context != b.KubeClient.CurrentContext() && kubectl.IsLocalKubernetes(cache.LastContext.Context) {
		mustRebuild = true
	}

	return mustRebuild, nil
}
