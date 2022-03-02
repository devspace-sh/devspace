package commands

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/strvals"
	"github.com/pkg/errors"
	"strings"
)

// BuildOptions describe how images should be build
type BuildOptions struct {
	build.Options

	Set       []string `long:"set" description:"Set configuration"`
	SetString []string `long:"set-string" description:"Set configuration as string"`
	From      []string `long:"from" description:"Reuse an existing configuration"`

	All bool `long:"all" description:"Build all images"`

	// Extra flags here to add an image
}

func Build(ctx *devspacecontext.Context, pipeline types.Pipeline, args []string) error {
	options := &BuildOptions{
		Options: pipeline.Options().BuildOptions,
	}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	ctx = ctx.WithConfig(ctx.Config.WithParsedConfig(ctx.Config.Config().Clone()))
	if options.All {
		for image := range ctx.Config.Config().Images {
			err = applyImageSetValues(ctx.Config.Config(), image, options.Set, options.SetString, options.From)
			if err != nil {
				return err
			}
		}
	} else if len(args) > 0 {
		for _, image := range args {
			err = applyImageSetValues(ctx.Config.Config(), image, options.Set, options.SetString, options.From)
			if err != nil {
				return err
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

func applyImageSetValues(config *latest.Config, image string, set, setString, from []string) error {
	if config.Images == nil {
		config.Images = map[string]*latest.Image{}
	}

	mapObj, err := applySetValues(image, set, setString, from, func(name string, create bool) (interface{}, error) {
		imageObj, ok := config.Images[image]
		if !ok {
			if !create {
				return nil, fmt.Errorf("couldn't find --from %s", name)
			}

			return &latest.Image{}, nil
		}

		return imageObj, nil
	})
	if err != nil {
		return err
	}

	imageObj := &latest.Image{}
	err = util.Convert(mapObj, imageObj)
	if err != nil {
		return err
	}

	config.Images[image] = imageObj
	return loader.Validate(config)
}

func applySetValues(name string, set, setString, from []string, getter func(name string, create bool) (interface{}, error)) (map[string]interface{}, error) {
	fromObj, err := getter(name, true)
	if err != nil {
		return nil, err
	}

	mapObj := map[string]interface{}{}
	err = util.Convert(fromObj, mapObj)
	if err != nil {
		return nil, err
	}

	for _, f := range from {
		obj, err := getter(f, false)
		if err != nil {
			return nil, err
		}

		getObj := map[string]interface{}{}
		err = util.Convert(obj, getObj)
		if err != nil {
			return nil, err
		}

		mapObj = strvals.MergeMaps(mapObj, getObj)
	}

	for _, s := range set {
		err = strvals.ParseInto(s, mapObj)
		if err != nil {
			return nil, errors.Wrap(err, "parsing --set flag")
		}
	}

	for _, s := range setString {
		err = strvals.ParseInto(s, mapObj)
		if err != nil {
			return nil, errors.Wrap(err, "parsing --set-string flag")
		}
	}

	return mapObj, nil
}
