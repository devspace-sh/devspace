package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// UpgradeCmd is a struct that defines a command call for "upgrade"
type UpgradeCmd struct{}

// NewUpgradeCmd creates a new upgrade command
func NewUpgradeCmd() *cobra.Command {
	cmd := &UpgradeCmd{}

	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the DevSpace CLI to the newest version",
		Long: `
#######################################################
################## devspace upgrade ###################
#######################################################
Upgrades the DevSpace CLI to the newest version
#######################################################`,
		Args: cobra.NoArgs,
		RunE: cmd.Run,
	}

	return upgradeCmd
}

// Run executes the command logic
func (cmd *UpgradeCmd) Run(cobraCmd *cobra.Command, args []string) error {
	err := upgrade.Upgrade()
	if err != nil {
		return errors.Errorf("Couldn't upgrade: %v", err)
	}

	return nil
}
