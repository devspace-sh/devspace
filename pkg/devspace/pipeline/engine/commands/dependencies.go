package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/pkg/errors"
)

// DependencyOptions describe how dependencies should get deployed
type DependencyOptions struct {
	All bool `long:"all" description:"Deploy all dependencies"`
}

func Dependency(ctx *devspacecontext.Context, dependencyRegistry registry.DependencyRegistry, args []string) error {
	options := &DependencyOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	if !options.All && len(args) == 0 {
		return fmt.Errorf("either specify 'run_dependency_pipelines --all' or 'run_dependency_pipelines dep1 dep2'")
	}
	return nil
}
