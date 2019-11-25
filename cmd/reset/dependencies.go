package reset

import (
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type dependenciesCmd struct {
}

func newDependenciesCmd() *cobra.Command {
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
		RunE: cmd.RunResetDependencies,
	}

	return dependenciesCmd
}

// RunResetDependencies executes the reset dependencies command logic
func (cmd *dependenciesCmd) RunResetDependencies(cobraCmd *cobra.Command, args []string) error {
	err := os.RemoveAll(dependency.DependencyFolderPath)
	if err != nil {
		return errors.Wrapf(err, "delete %s", dependency.DependencyFolderPath)
	}

	log.GetInstance().Done("Successfully reseted the dependency cache")
	return nil
}
