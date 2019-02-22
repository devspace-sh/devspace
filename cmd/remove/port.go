package remove

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type portCmd struct {
	LabelSelector string
	RemoveAll     bool
}

func newPortCmd() *cobra.Command {
	cmd := &portCmd{}

	portCmd := &cobra.Command{
		Use:   "port",
		Short: "Removes forwarded ports from a devspace",
		Long: `
	#######################################################
	############### devspace remove port ##################
	#######################################################
	Removes port mappings from the devspace configuration:
	devspace remove port 8080,3000
	devspace remove port --label-selector=release=test
	devspace remove port --all
	#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemovePort,
	}

	portCmd.Flags().StringVar(&cmd.LabelSelector, "label-selector", "", "Comma separated key=value selector list (e.g. release=test)")
	portCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all configured ports")

	return portCmd
}

// RunRemovePort executes the remove port command logic
func (cmd *portCmd) RunRemovePort(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	err = configure.RemovePort(cmd.RemoveAll, cmd.LabelSelector, args)
	if err != nil {
		log.Fatal(err)
	}
}
