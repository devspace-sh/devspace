package custom

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/util/shell"
	"io"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/hash"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	dockerterm "github.com/moby/term"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	_, stdout, _ = dockerterm.StdStreams()
)

// Builder holds all the relevant information for a custom build
type Builder struct {
	imageConf *latest.ImageConfig

	imageConfigName string
	imageTags       []string

	config       config.Config
	dependencies []types.Dependency
}

// NewBuilder creates a new custom builder
func NewBuilder(imageConfigName string, imageConf *latest.ImageConfig, imageTags []string, config config.Config, dependencies []types.Dependency) *Builder {
	return &Builder{
		imageConfigName: imageConfigName,
		imageConf:       imageConf,
		imageTags:       imageTags,

		config:       config,
		dependencies: dependencies,
	}
}

// ShouldRebuild implements interface
func (b *Builder) ShouldRebuild(cache *generated.CacheConfig, forceRebuild bool, log logpkg.Logger) (bool, error) {
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
		files, err := doublestar.Glob(pattern)
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
	mustRebuild := forceRebuild || b.imageConf.RebuildStrategy == latest.RebuildStrategyAlways || imageCache.Tag == "" || imageCache.ImageConfigHash != imageConfigHash || imageCache.CustomFilesHash != customFilesHash

	imageCache.ImageConfigHash = imageConfigHash
	imageCache.CustomFilesHash = customFilesHash

	return mustRebuild, nil
}

// Build implements interface
func (b *Builder) Build(devspacePID string, log logpkg.Logger) error {
	// Build arguments
	args := []string{}

	// resolve command
	if len(b.imageTags) > 0 {
		key := fmt.Sprintf("images.%s", b.imageConfigName)
		b.config.SetRuntimeVariable(key, b.imageConf.Image+":"+b.imageTags[0])
		b.config.SetRuntimeVariable(key+".image", b.imageConf.Image)
		b.config.SetRuntimeVariable(key+".tag", b.imageTags[0])
	}
	
	// loop over args
	for i := range b.imageConf.Build.Custom.Args {
		resolvedArg, err := runtime.NewRuntimeResolver(false).FillRuntimeVariablesAsString(b.imageConf.Build.Custom.Args[i], b.config, b.dependencies)
		if err != nil {
			return err
		}
		
		args = append(args, resolvedArg)
	}

	// add image arg
	if !b.imageConf.Build.Custom.SkipImageArg {
		for _, tag := range b.imageTags {
			if b.imageConf.Build.Custom.ImageFlag != "" {
				args = append(args, b.imageConf.Build.Custom.ImageFlag)
			}

			if !b.imageConf.Build.Custom.ImageTagOnly {
				args = append(args, b.imageConf.Image+":"+tag)
			} else {
				args = append(args, tag)
			}
		}
	}

	// append the rest
	for i := range b.imageConf.Build.Custom.AppendArgs {
		resolvedArg, err := runtime.NewRuntimeResolver(false).FillRuntimeVariablesAsString(b.imageConf.Build.Custom.AppendArgs[i], b.config, b.dependencies)
		if err != nil {
			return err
		}

		args = append(args, resolvedArg)
	}

	// get the command
	commandPath := b.imageConf.Build.Custom.Command
	for _, c := range b.imageConf.Build.Custom.Commands {
		if !command.ShouldExecuteOnOS(c.OperatingSystem) {
			continue
		}

		commandPath = c.Command
		break
	}
	if commandPath == "" {
		return fmt.Errorf("no command specified for custom builder")
	}

	// resolve command and args
	commandPath, err := runtime.NewRuntimeResolver(false).FillRuntimeVariablesAsString(commandPath, b.config, b.dependencies)
	if err != nil {
		return err
	}

	// Determine output writer
	var writer io.Writer
	if log == logpkg.GetInstance() {
		writer = stdout
	} else {
		writer = log
	}

	log.Infof("Build %s:%s with custom command '%s %s'", b.imageConf.Image, b.imageTags[0], commandPath, strings.Join(args, " "))

	if len(args) == 0 {
		err = shell.ExecuteShellCommand(commandPath, args, filepath.Dir(b.config.Path()), writer, writer, nil)
		if err != nil {
			return errors.Errorf("error building image: %v", err)
		}
	} else {
		err = command.NewStreamCommand(commandPath, args).Run(writer, writer, nil)
		if err != nil {
			return errors.Errorf("error building image: %v", err)
		}
	}

	log.Done("Done processing image '" + b.imageConf.Image + "'")
	return nil
}
