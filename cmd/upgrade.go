package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// UpgradeCmd is a struct that defines a command call for "upgrade"
type UpgradeCmd struct {
	flags *UpgradeCmdFlags
}

// UpgradeCmdFlags are the flags available for the upgrade-command
type UpgradeCmdFlags struct {
}

func init() {
	cmd := &UpgradeCmd{
		flags: &UpgradeCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the DevSpace.cli to the newest version",
		Long: `
#######################################################
################## devspace upgrade ###################
#######################################################
Upgrades the DevSpace.cli to the newest version
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

// Run executes the command logic
func (cmd *UpgradeCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()
	err := upgrade.Upgrade()

	if err != nil {
		log.Fatalf("Couldn't upgrade: %s", err.Error())
	}
}
