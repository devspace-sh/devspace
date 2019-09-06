package update

import (
	"context"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/spf13/cobra"
)

// dependenciesCmd holds the cmd flags
type dependenciesCmd struct {
	AllowCyclicDependencies bool
}

// newDependenciesCmd creates a new command
func newDependenciesCmd() *cobra.Command {
	cmd := &dependenciesCmd{}

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
		Run:  cmd.RunDependencies,
	}

	dependenciesCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")

	return dependenciesCmd
}

// RunDependencies executes the functionality "devspace update dependencies"
func (cmd *dependenciesCmd) RunDependencies(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Get the config
	config := configutil.GetConfig(context.Background())

	// Load generated config
	generatedConfig, err := generated.LoadConfig(context.Background())
	if err != nil {
		log.Fatalf("Error loading generated.yaml: %v", err)
	}

	err = dependency.UpdateAll(config, generatedConfig, cmd.AllowCyclicDependencies, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully updated all dependencies")
}
