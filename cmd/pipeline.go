package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// PipelineCmd is a struct that defines a command call for "print"
type PipelineCmd struct {
	*flags.GlobalFlags
}

// NewPipelineCmd creates a new devspace pipeline command
func NewPipelineCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PipelineCmd{
		GlobalFlags: globalFlags,
	}

	pipelineCmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Runs the specified pipeline",
		Long: `
#######################################################
################ devspace pipeline ####################
#######################################################
Runs the specified pipeline
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	return pipelineCmd
}

// Run executes the command logic
func (cmd *PipelineCmd) Run(f factory.Factory) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions(log)
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		log.Warnf("Unable to create new kubectl client: %v", err)
	}
	configOptions.KubeClient = client

	// load config
	loadedConfig, err := configLoader.Load(configOptions, log)
	if err != nil {
		return err
	}

	// resolve dependencies
	dependencies, err := dependency.NewManager(loadedConfig, client, configOptions, log).ResolveAll(dependency.ResolveOptions{
		Silent: true,
	})
	if err != nil {
		log.Warnf("Error resolving dependencies: %v", err)
	}

	return pipeline.NewExecutor(loadedConfig, dependencies, client).ExecutePipeline(loadedConfig.Config().Pipeline, log)
}
