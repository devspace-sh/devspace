package add

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type portCmd struct {
	*flags.GlobalFlags

	LabelSelector string
}

func newPortCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &portCmd{GlobalFlags: globalFlags}

	addPortCmd := &cobra.Command{
		Use:   "port",
		Short: "Add a new port forward configuration",
		Long: `
#######################################################
################ devspace add port ####################
#######################################################
Add a new port mapping to your DevSpace configuration
(format is local:remote comma separated):
devspace add port 8080:80,3000
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddPort,
	}

	addPortCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Comma separated key=value label-selector list (e.g. release=test)")

	return addPortCmd
}

// RunAddPort executes the add port command logic
func (cmd *portCmd) RunAddPort(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	config := configutil.GetBaseConfig(cmd.KubeContext)

	err = configure.AddPort(config, cmd.Namespace, cmd.LabelSelector, args)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully added port %v", args[0])
}
