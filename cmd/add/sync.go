package add

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type syncCmd struct {
	LabelSelector string
	LocalPath     string
	ContainerPath string
	ExcludedPaths string
	Namespace     string
	Service       string
}

func newSyncCmd() *cobra.Command {
	cmd := &syncCmd{}

	addSyncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Add a sync path to the devspace",
		Long: `
#######################################################
################# devspace add sync ###################
#######################################################
Add a sync path to the devspace

How to use:
devspace add sync --local=app --container=/app
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunAddSync,
	}

	addSyncCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Comma separated key=value selector list (e.g. release=test)")
	addSyncCmd.Flags().StringVar(&cmd.LocalPath, "local", "", "Relative local path")
	addSyncCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "Namespace to use")
	addSyncCmd.Flags().StringVar(&cmd.ContainerPath, "container", "", "Absolute container path")
	addSyncCmd.Flags().StringVar(&cmd.ExcludedPaths, "exclude", "", "Comma separated list of paths to exclude (e.g. node_modules/,bin,*.exe)")
	addSyncCmd.Flags().StringVar(&cmd.Service, "service", "", "The devspace config service")

	addSyncCmd.MarkFlagRequired("local")
	addSyncCmd.MarkFlagRequired("container")

	return addSyncCmd
}

// RunAddSync executes the add sync command logic
func (cmd *syncCmd) RunAddSync(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	err = configure.AddSyncPath(cmd.LocalPath, cmd.ContainerPath, cmd.Namespace, cmd.LabelSelector, cmd.ExcludedPaths, cmd.Service)
	if err != nil {
		log.Fatalf("Error adding sync path: %v", err)
	}

	log.Donef("Successfully added sync between local path %v and container path %v", cmd.LocalPath, cmd.ContainerPath)
}
