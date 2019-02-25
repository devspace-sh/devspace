package use

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type registryCmd struct {
}

func newRegistryCmd() *cobra.Command {
	cmd := &registryCmd{}

	registryCmd := &cobra.Command{
		Use:   "registry",
		Short: "Configure docker to use a specific registry",
		Long: `
#######################################################
############### devspace use registry #################
#######################################################
Define which registry to use.

Example:
devspace use registry dscr.io
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunUseRegistry,
	}

	return registryCmd
}

// RunUseRegistry executes the functionality "devspace use registry"
func (cmd *registryCmd) RunUseRegistry(cobraCmd *cobra.Command, args []string) {
	// Get provider
	provider, err := cloud.GetCurrentProvider(log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Login
	err = provider.LoginIntoRegistry(args[0])
	if err != nil {
		log.Fatalf("Error loging into registry %s: %v", args[0], err)
	}

	log.Infof("Successfully logged into registry %s", args[0])
}
