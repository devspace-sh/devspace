package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/upgrade"
	"github.com/spf13/cobra"
)

type UpgradeCmd struct {
	flags *UpgradeCmdFlags
}

type UpgradeCmdFlags struct {
}

func init() {
	cmd := &UpgradeCmd{
		flags: &UpgradeCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the devspace cli to the newest version",
		Long: `
#######################################################
################## devspace upgrade ###################
#######################################################
Upgrades the devspace cli to the newest version
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

func (cmd *UpgradeCmd) Run(cobraCmd *cobra.Command, args []string) {
	err := upgrade.Upgrade()

	if err != nil {
		log.Fatalf("Couldn't upgrade: %s\n", err.Error())
	}
}
