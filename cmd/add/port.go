package add

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type portCmd struct {
	LabelSelector string
	Namespace     string
	Service       string
}

func newPortCmd() *cobra.Command {
	cmd := &portCmd{}

	addPortCmd := &cobra.Command{
		Use:   "port",
		Short: "Add a new port forward configuration",
		Long: `
#######################################################
################ devspace add port ####################
#######################################################
Add a new port mapping that should be forwarded to
the devspace (format is local:remote comma separated):
devspace add port 8080:80,3000
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddPort,
	}

	addPortCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "Namespace to use")
	addPortCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Comma separated key=value label-selector list (e.g. release=test)")
	addPortCmd.Flags().StringVar(&cmd.Service, "service", "", "The devspace config service")

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
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	err = configure.AddPort(cmd.Namespace, cmd.LabelSelector, cmd.Service, args)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully added port %v", args[0])
}
