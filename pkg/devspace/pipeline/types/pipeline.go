package types

import (
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	types2 "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
)

type Options struct {
	BuildOptions      build.Options
	DeployOptions     deploy.Options
	DependencyOptions DependencyOptions
	DevOptions        devpod.Options
}

type DependencyOptions struct {
	Pipeline   string   `long:"pipeline" description:"The pipeline to deploy from the dependency"`
	Exclude    []string `long:"exclude" description:"Dependencies to exclude"`
	Sequential bool     `long:"sequential" description:"Run dependencies one after another"`
}

type Pipeline interface {
	// Run runs the main pipeline
	Run(ctx *devspacecontext.Context) error

	// DevPodManager retrieves the used dev pod manager
	DevPodManager() devpod.Manager

	// DependencyRegistry retrieves the dependency registry
	DependencyRegistry() registry.DependencyRegistry

	// Dependencies retrieves the currently created dependencies
	Dependencies() []Pipeline

	// Options retrieves the default options for the pipeline
	Options() Options

	// Name retrieves the name of the pipeline
	Name() string

	// WaitDev waits for the dependency dev managers as well current
	// dev pod manager to be finished
	WaitDev()

	// StartNewPipelines starts sub pipelines in this pipeline. It is ensured
	// that each pipeline can only be run once at the same time and otherwise
	// will fail to start.
	StartNewPipelines(ctx *devspacecontext.Context, pipelines []*latest.Pipeline, sequentially bool) error

	// StartNewDependencies starts dependency pipelines in this pipeline. It is ensured
	// that each pipeline will only run once ever and will otherwise be skipped.
	StartNewDependencies(ctx *devspacecontext.Context, dependencies []types2.Dependency, options DependencyOptions) error
}
