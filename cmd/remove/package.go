package remove

import (
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type packageCmd struct {
	RemoveAll  bool
	Deployment string
}

func newPackageCmd() *cobra.Command {
	cmd := &packageCmd{}

	packageCmd := &cobra.Command{
		Use:   "package",
		Short: "Removes one or all packages from a devspace",
		Long: `
	#######################################################
	############## devspace remove package ################
	#######################################################
	Removes a package from the devspace:
	devspace remove package mysql
	devspace remove package mysql -d devspace-default
	#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemovePackage,
	}

	packageCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all packages")
	packageCmd.Flags().StringVarP(&cmd.Deployment, "deployment", "d", "", "The deployment name to use")

	return packageCmd
}

// RunRemovePackage executes the remove package command logic
func (cmd *packageCmd) RunRemovePackage(cobraCmd *cobra.Command, args []string) {
	err := configure.RemovePackage(cmd.RemoveAll, cmd.Deployment, args, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}
}
