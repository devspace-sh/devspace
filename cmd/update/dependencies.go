package update

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/spf13/cobra"
)

// dependenciesCmd holds the cmd flags
type dependenciesCmd struct {
	*flags.GlobalFlags

	AllowCyclicDependencies bool
}

// newDependenciesCmd creates a new command
func newDependenciesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
	config := configutil.GetConfig(cmd.KubeContext, cmd.Profile)

	// Load generated config
	generatedConfig, err := generated.LoadConfig(cmd.Profile)
	if err != nil {
		log.Fatalf("Error loading generated.yaml: %v", err)
	}

	err = dependency.UpdateAll(config, generatedConfig, cmd.AllowCyclicDependencies, cmd.KubeContext, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully updated all dependencies")
}
