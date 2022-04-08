package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/interp"
	"strings"
)

type GetImageOptions struct {
	Dependency string `long:"dependency" description:"Retrieves the image from the named dependency"`
	Only       string `long:"only" description:"Displays either only the tag or only the image"`
}

func GetImage(ctx devspacecontext.Context, args []string) error {
	ctx = ctx.WithLogger(ctx.Log().ErrorStreamOnly())
	ctx.Log().Debugf("get_image %s", strings.Join(args, " "))
	options := &GetImageOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: get_image [--only=image|tag] [--dependency=DEPENDENCY] [image_name]")
	}

	var (
		onlyImage bool
		onlyTag   bool
	)
	switch options.Only {
	case "":
	case "image":
		onlyImage = true
	case "tag":
		onlyTag = true
	default:
		return fmt.Errorf("usage: get_image [--only=image|tag] [--dependency=DEPENDENCY] [image_name]")
	}

	if options.Dependency != "" {
		found := false
		for _, dep := range ctx.Dependencies() {
			if dep.Name() == options.Dependency {
				ctx = ctx.AsDependency(dep)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("couldn't find dependency %v", options.Dependency)
		}
	}

	_, imageCache, err := runtime.GetImage(ctx.Config(), args[0], onlyImage, onlyTag)
	if err != nil {
		return err
	}

	hc := interp.HandlerCtx(ctx.Context())
	_, _ = hc.Stdout.Write([]byte(imageCache))
	return nil
}
