package types

import (
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	types2 "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"mvdan.cc/sh/v3/expand"
)

type Options struct {
	BuildOptions      build.Options
	DeployOptions     deploy.Options
	PurgeOptions      deploy.PurgeOptions
	DependencyOptions DependencyOptions
	DevOptions        devpod.Options
}

type DependencyOptions struct {
	Pipeline   string   `long:"pipeline" description:"The pipeline to deploy from the dependency"`
	Exclude    []string `long:"exclude" description:"Dependencies to exclude"`
	Only       []string `long:"only" description:"Dependencies to include"`
	Sequential bool     `long:"sequential" description:"Run dependencies one after another"`

	SetFlag []string `long:"set-flag" description:"Set a pipeline flag"`
}

// PipelineOptions describe how pipelines should be run
type PipelineOptions struct {
	Background bool     `long:"background" description:"Run the pipeline in the background"`
	Sequential bool     `long:"sequential" description:"Run pipelines one after another"`
	SetFlag    []string `long:"set-flag" description:"Set a pipeline flag"`

	Environ expand.Environ
}

type Pipeline interface {
	// Run runs the main pipeline
	Run(ctx devspacecontext.Context, args []string) error

	// DevPodManager retrieves the used dev pod manager
	DevPodManager() devpod.Manager

	// DependencyRegistry retrieves the dependency registry
	DependencyRegistry() registry.DependencyRegistry

	// Parent retrieves the pipeline parent or nil if there is none
	Parent() Pipeline

	// Exclude locks the upmost parent pipeline in the namespace and ensures only
	// a single DevSpace instance is running this project at the same time
	Exclude(ctx devspacecontext.Context) error

	// Dependencies retrieves the currently created dependencies
	Dependencies() map[string]Pipeline

	// Close kills the pipeline including all dependencies and waits for it
	// to exit as well as closes the dev pod manager and all related dev pods
	Close() error

	// Options retrieves the default options for the pipeline
	Options() Options

	// Name retrieves the name of the DevSpace yaml. This is NOT the name of the
	// pipeline like deploy, dev or purge and holds the value of the current
	// project like my-microservice etc.
	Name() string

	// Done returns a channel that is closed when the pipeline is done running
	Done() <-chan struct{}

	// WaitDev waits for the dependency dev managers as well current
	// dev pod manager to be finished
	WaitDev() error

	// StartNewPipelines starts sub pipelines in this pipeline. It is ensured
	// that each pipeline can only be run once at the same time and otherwise
	// will fail to start.
	StartNewPipelines(ctx devspacecontext.Context, pipelines []*latest.Pipeline, options PipelineOptions) error

	// StartNewDependencies starts dependency pipelines in this pipeline. It is ensured
	// that each pipeline will only run once ever and will otherwise be skipped.
	StartNewDependencies(ctx devspacecontext.Context, dependencies []types2.Dependency, options DependencyOptions) error
}
