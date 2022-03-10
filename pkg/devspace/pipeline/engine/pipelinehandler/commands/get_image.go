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
	Only string `long:"only" description:"Displays either only the tag or only the image"`
}

func GetImage(ctx *devspacecontext.Context, args []string) error {
	ctx.Log.Debugf("get_image %s", strings.Join(args, " "))
	options := &GetImageOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: get_image [image_name] [--only=image|tag]")
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
		return fmt.Errorf("usage: get_image [image_name] [--only=image|tag]")
	}

	_, imageCache, err := runtime.GetImage(ctx.Config, args[0], onlyImage, onlyTag)
	if err != nil {
		return err
	}

	hc := interp.HandlerCtx(ctx.Context)
	_, _ = hc.Stdout.Write([]byte(imageCache))
	return nil
}
