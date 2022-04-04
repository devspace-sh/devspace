package commands

import (
	"fmt"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/pkg/errors"
)

// BuildImagesOptions describe how images should be build
type BuildImagesOptions struct {
	build.Options

	All bool `long:"all" description:"Build all images"`

	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
	From      []string `long:"from" description:"Reuse an existing configuration"`
	FromFile  []string `long:"from-file" description:"Reuse an existing configuration from a file"`
}

func BuildImages(ctx *devspacecontext.Context, pipeline types.Pipeline, args []string) error {
	ctx.Log.Debugf("build_images %s", strings.Join(args, " "))
	options := &BuildImagesOptions{
		Options: pipeline.Options().BuildOptions,
	}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if options.All {
		images := ctx.Config.Config().Images
		for image := range images {
			ctx, err = applySetValues(ctx, "images", image, options.Set, options.SetString, options.From, options.FromFile)
			if err != nil {
				return err
			}
		}
	} else if len(args) > 0 {
		for _, image := range args {
			ctx, err = applySetValues(ctx, "images", image, options.Set, options.SetString, options.From, options.FromFile)
			if err != nil {
				return err
			}
			if ctx.Config.Config().Images == nil || ctx.Config.Config().Images[image] == nil {
				return fmt.Errorf("couldn't find image %v", image)
			}
		}
	} else {
		return fmt.Errorf("either specify 'build_images --all' or 'build_images image1 image2'")
	}

	err = build.NewController().Build(ctx, args, &options.Options)
	if err != nil {
		if strings.Contains(err.Error(), "no space left on device") {
			return errors.Errorf("Error building image: %v\n\n Try running `docker system prune` to free docker daemon space and retry", err)
		}

		return errors.Wrap(err, "build images")
	}

	return nil
}
