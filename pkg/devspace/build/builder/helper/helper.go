package helper

import (
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"os"
)

// BuildHelper is the helper class to store common functionality used by both the docker and kaniko builder
type BuildHelper struct {
	ImageConfigName string
	ImageConf       *latest.Image

	DockerfilePath string
	ContextPath    string

	EngineName string
	ImageName  string
	ImageTags  []string
	Entrypoint []string
	Cmd        []string
}

// BuildHelperInterface is the interface the build helper uses to build an image
type BuildHelperInterface interface {
	BuildImage(ctx devspacecontext.Context, absoluteContextPath string, absoluteDockerfilePath string, entrypoint []string, cmd []string) error
}

// NewBuildHelper creates a new build helper for a certain engine
func NewBuildHelper(ctx devspacecontext.Context, engineName string, imageConfigName string, imageConf *latest.Image, imageTags []string) *BuildHelper {
	var (
		dockerfilePath, contextPath = GetDockerfileAndContext(ctx, imageConf)
		imageName                   = imageConf.Image
	)

	// Check if we should overwrite entrypoint
	var (
		entrypoint []string
		cmd        []string
	)

	if imageConf.Entrypoint != nil {
		entrypoint = imageConf.Entrypoint
	}
	if imageConf.Cmd != nil {
		cmd = imageConf.Cmd
	}

	return &BuildHelper{
		ImageConfigName: imageConfigName,
		ImageConf:       imageConf,

		DockerfilePath: dockerfilePath,
		ContextPath:    contextPath,

		ImageName:  imageName,
		ImageTags:  imageTags,
		EngineName: engineName,

		Entrypoint: entrypoint,
		Cmd:        cmd,
	}
}

// Build builds a new image
func (b *BuildHelper) Build(ctx devspacecontext.Context, imageBuilder BuildHelperInterface) error {
	ctx.Log().Infof("Building image '%s:%s' with engine '%s'", b.ImageName, b.ImageTags[0], b.EngineName)

	// Build Image
	err := imageBuilder.BuildImage(ctx, b.ContextPath, b.DockerfilePath, b.Entrypoint, b.Cmd)
	if err != nil {
		return err
	}

	ctx.Log().Done("Done processing image '" + b.ImageName + "'")
	return nil
}

// ShouldRebuild determines if the image should be rebuilt
func (b *BuildHelper) ShouldRebuild(ctx devspacecontext.Context, forceRebuild bool) (bool, error) {
	imageCache, _ := ctx.Config().LocalCache().GetImageCache(b.ImageConfigName)

	// if rebuild strategy is always, we return here
	if b.ImageConf.RebuildStrategy == latest.RebuildStrategyAlways {
		ctx.Log().Infof("Rebuild image %s because strategy is always rebuild", imageCache.ImageName)
		return true, nil
	}

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
	if imageCache.Tag == "" {
		ctx.Log().Infof("Rebuild image %s because tag is missing", imageCache.ImageName)
	} else if imageCache.DockerfileHash != dockerfileHash {
		ctx.Log().Infof("Rebuild image %s because dockerfile has changed", imageCache.ImageName)
	} else if imageCache.ImageConfigHash != imageConfigHash {
		ctx.Log().Infof("Rebuild image %s because image config has changed", imageCache.ImageName)
	} else if imageCache.EntrypointHash != entrypointHash {
		ctx.Log().Infof("Rebuild image %s because entrypoint has changed", imageCache.ImageName)
	}

	// Okay this check verifies if the previous deploy context was local kubernetes context where we didn't push the image and now have a kubernetes context where we probably push
	// or use another docker client (e.g. minikube <-> docker-desktop)
	if !mustRebuild && ctx.KubeClient() != nil && ctx.Config().LocalCache().GetLastContext() != nil && ctx.Config().LocalCache().GetLastContext().Context != ctx.KubeClient().CurrentContext() && kubectl.IsLocalKubernetes(ctx.Config().LocalCache().GetLastContext().Context) {
		mustRebuild = true
		ctx.Log().Infof("Rebuild image %s because previous build was local kubernetes", imageCache.ImageName)
		ctx.Config().LocalCache().SetLastContext(&localcache.LastContextConfig{
			Namespace: ctx.KubeClient().Namespace(),
			Context:   ctx.KubeClient().CurrentContext(),
		})
	}

	// Check if should consider context path changes for rebuilding
	if b.ImageConf.RebuildStrategy != latest.RebuildStrategyIgnoreContextChanges {
		// Hash context path
		contextDir, relDockerfile, err := build.GetContextFromLocalDir(b.ContextPath, b.DockerfilePath)
		if err != nil {
			return false, errors.Wrap(err, "get context from local dir")
		}

		relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)
		excludes, err := ReadDockerignore(contextDir, relDockerfile)
		if err != nil {
			return false, errors.Errorf("Error reading .dockerignore: %v", err)
		}

		contextHash, err := hash.DirectoryExcludes(contextDir, excludes, false)
		if err != nil {
			return false, errors.Errorf("Error hashing %s: %v", contextDir, err)
		}

		if !mustRebuild && imageCache.ContextHash != contextHash {
			ctx.Log().Infof("Rebuild image %s because build context has changed", imageCache.ImageName)
		}
		mustRebuild = mustRebuild || imageCache.ContextHash != contextHash

		// TODO: This is not an ideal solution since there can be the issue that the user runs
		// devspace dev & the generated.yaml is written without ContextHash and on a subsequent
		// devspace deploy the image would be rebuild, because the ContextHash was empty and is
		// now different. However in this case it is probably better to save the context hash computing
		// time during devspace dev instead of always hashing the context path.
		if forceRebuild || mustRebuild {
			imageCache.ContextHash = contextHash
		}
	}

	if forceRebuild || mustRebuild {
		imageCache.DockerfileHash = dockerfileHash
		imageCache.ImageConfigHash = imageConfigHash
		imageCache.EntrypointHash = entrypointHash
	}

	ctx.Config().LocalCache().SetImageCache(b.ImageConfigName, imageCache)
	return mustRebuild, nil
}

func (b *BuildHelper) IsImageAvailableLocally(ctx devspacecontext.Context, dockerClient dockerclient.Client) (bool, error) {
	// Hack to check if docker is present in the system
	// if docker is not present then skip the image availability check
	// and return (true, nil) to skip image rebuild
	// if docker is present then do the image availability check
	err := command.Command(ctx.Context(), ctx.WorkingDir(), ctx.Environ(), nil, nil, nil, "docker", "buildx")
	if err != nil {
		return true, nil
	}

	imageCache, _ := ctx.Config().LocalCache().GetImageCache(b.ImageConfigName)
	imageName := imageCache.ImageName + ":" + imageCache.Tag
	dockerAPIClient := dockerClient.DockerAPIClient()
	imageList, err := dockerAPIClient.ImageList(ctx.Context(), types.ImageListOptions{})
	if err != nil {
		return false, err
	}
	for _, image := range imageList {
		for _, repoTag := range image.RepoTags {
			if repoTag == imageName {
				return true, nil
			}
		}
	}
	return false, nil
}
