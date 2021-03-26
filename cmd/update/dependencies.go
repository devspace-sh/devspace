package update

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// dependenciesCmd holds the cmd flags
type dependenciesCmd struct {
	*flags.GlobalFlags

	AllowCyclicDependencies bool
}

// newDependenciesCmd creates a new command
func newDependenciesCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &dependenciesCmd{GlobalFlags: globalFlags}

	dependenciesCmd := &cobra.Command{
		Use:   "dependencies",
		Short: "Updates the git repositories of the dependencies defined in the devspace.yaml",
		Long: `
#######################################################
############ devspace update dependencies #############
#######################################################
Updates the git repositories of the dependencies defined
in the devspace.yaml
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunDependencies(f, cobraCmd, args)
		},
	}

	dependenciesCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")

	return dependenciesCmd
}

// RunDependencies executes the functionality "devspace update dependencies"
func (cmd *dependenciesCmd) RunDependencies(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Get the config
	config, err := configLoader.Load(configOptions, log)
	if err != nil {
		return err
	}

	// Load generated config
	generatedConfig := config.Generated()

	// Create Dependencymanager
	manager, err := dependency.NewManager(config.Config(), generatedConfig, nil, cmd.AllowCyclicDependencies, configOptions, log)
	if err != nil {
		return errors.Wrap(err, "new manager")
	}

	err = manager.UpdateAll()
	if err != nil {
		return err
	}

	log.Donef("Successfully updated all dependencies")
	return nil
}
