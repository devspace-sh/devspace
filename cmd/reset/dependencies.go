package reset

import (
	"os"

	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"

	"github.com/loft-sh/devspace/pkg/util/factory"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type dependenciesCmd struct {
}

func newDependenciesCmd(f factory.Factory) *cobra.Command {
	cmd := &dependenciesCmd{}

	dependenciesCmd := &cobra.Command{
		Use:   "dependencies",
		Short: "Resets the dependencies cache",
		Long: `
#######################################################
############ devspace reset dependencies ##############
#######################################################
Deletes the complete dependency cache

Examples:
devspace reset dependencies
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunResetDependencies(f, cobraCmd, args)
		}}

	return dependenciesCmd
}

// RunResetDependencies executes the reset dependencies command logic
func (cmd *dependenciesCmd) RunResetDependencies(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	log := f.GetLog()
	err := os.RemoveAll(dependencyutil.DependencyFolderPath)
	if err != nil {
		return errors.Wrapf(err, "delete %s", dependencyutil.DependencyFolderPath)
	}

	log.Done("Successfully reseted the dependency cache")
	return nil
}
