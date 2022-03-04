package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	types2 "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/pkg/errors"
)

// DependencyOptions describe how dependencies should get deployed
type DependencyOptions struct {
	types.DependencyOptions

	All bool `long:"all" description:"Deploy all dependencies"`
}

func Dependency(ctx *devspacecontext.Context, pipeline types.Pipeline, args []string) error {
	options := &DependencyOptions{
		DependencyOptions: pipeline.Options().DependencyOptions,
	}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	duplicates := map[string]bool{}
	deployDependencies := []types2.Dependency{}
	if options.All {
		deployDependencies = ctx.Dependencies
	} else if len(args) > 0 {
		for _, arg := range args {
			if duplicates[arg] {
				continue
			}

			duplicates[arg] = true
			found := false
			for _, dep := range ctx.Dependencies {
				if dep.Name() == arg {
					deployDependencies = append(deployDependencies, dep)
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("couldn't find dependency %s", arg)
			}
		}
	} else {
		return fmt.Errorf("either specify 'run_dependency_pipelines --all' or 'run_dependency_pipelines dep1 dep2'")
	}

	return pipeline.StartNewDependencies(ctx, deployDependencies, options.DependencyOptions)
}
