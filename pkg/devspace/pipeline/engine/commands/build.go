package commands

import (
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/pkg/errors"
)

// BuildOptions describe how images should be build
type BuildOptions struct {
	build.Options

	All bool `long:"all" description:"Build all images"`

	// Extra flags here to add an image
}

func Build(ctx *devspacecontext.Context, args []string) error {
	options := &BuildOptions{}
	args, err := flags.ParseArgs(&options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if len(args) == 0 || (options.All && len(args) > 0) {
		return fmt.Errorf("build: either specify 'build --all' or 'build image1 image2'")
	}
	return build.NewController().Build(ctx, args, &options.Options)
}
