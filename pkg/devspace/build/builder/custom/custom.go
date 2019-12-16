package custom

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/command"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"

	dockerterm "github.com/docker/docker/pkg/term"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	_, stdout, stderr = dockerterm.StdStreams()
)

// Builder holds all the relevant information for a custom build
type Builder struct {
	imageConf *latest.ImageConfig

	imageConfigName string
	imageTag        string

	cmd command.Interface
}

// NewBuilder creates a new custom builder
func NewBuilder(imageConfigName string, imageConf *latest.ImageConfig, imageTag string) *Builder {
	return &Builder{
		imageConfigName: imageConfigName,
		imageConf:       imageConf,
		imageTag:        imageTag,
	}
}

// ShouldRebuild implements interface
func (b *Builder) ShouldRebuild(cache *generated.CacheConfig, ignoreContextPathChanges bool) (bool, error) {
	if b.imageConf.Build.Custom.OnChange == nil || len(b.imageConf.Build.Custom.OnChange) == 0 {
		return true, nil
	}

	// Hash image config
	configStr, err := yaml.Marshal(*b.imageConf)
	if err != nil {
		return false, errors.Wrap(err, "marshal image config")
	}
	imageConfigHash := hash.String(string(configStr))

	// Loop over on change globs
	customFilesHash := ""
	for _, pattern := range b.imageConf.Build.Custom.OnChange {
		files, err := doublestar.Glob(*pattern)
		if err != nil {
			return false, err
		}

		for _, file := range files {
			sha256, err := hash.Directory(file)
			if err != nil {
				return false, errors.Wrap(err, "hash "+file)
			}

			customFilesHash += sha256
		}
	}
	customFilesHash = hash.String(customFilesHash)

	imageCache := cache.GetImageCache(b.imageConfigName)

	// only rebuild Docker image when Dockerfile or context has changed since latest build
	mustRebuild := imageCache.Tag == "" || imageCache.ImageConfigHash != imageConfigHash || imageCache.CustomFilesHash != customFilesHash

	imageCache.ImageConfigHash = imageConfigHash
	imageCache.CustomFilesHash = customFilesHash

	return mustRebuild, nil
}

// Build implements interface
func (b *Builder) Build(log logpkg.Logger) error {
	// Build arguments
	args := []string{}

	if b.imageConf.Build.Custom.ImageFlag != "" {
		args = append(args, b.imageConf.Build.Custom.ImageFlag, b.imageConf.Image+":"+b.imageTag)
	} else {
		args = append(args, b.imageConf.Image+":"+b.imageTag)
	}

	if b.imageConf.Build.Custom.Args != nil {
		for _, arg := range b.imageConf.Build.Custom.Args {
			args = append(args, *arg)
		}
	}

	if b.cmd == nil {
		b.cmd = command.NewStreamCommand(filepath.FromSlash(b.imageConf.Build.Custom.Command), args)
	}

	// Determine output writer
	var writer io.Writer
	if log == logpkg.GetInstance() {
		writer = stdout
	} else {
		writer = log
	}

	log.Infof("Build %s:%s with custom command %s %s", b.imageConf.Image, b.imageTag, b.imageConf.Build.Custom.Command, strings.Join(args, " "))

	err := b.cmd.Run(writer, writer, nil)
	if err != nil {
		return errors.Errorf("Error building image: %v", err)
	}

	log.Done("Done processing image '" + b.imageConf.Image + "'")
	return nil
}
