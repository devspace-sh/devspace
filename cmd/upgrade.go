package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/upgrade"
	"github.com/covexo/devspace/pkg/util/log"
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

// Run executes the command logic
func (cmd *UpgradeCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()
	err := upgrade.Upgrade()

	if err != nil {
		log.Fatalf("Couldn't upgrade: %s", err.Error())
	}
}
