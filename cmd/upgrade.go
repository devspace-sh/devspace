package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// UpgradeCmd is a struct that defines a command call for "upgrade"
type UpgradeCmd struct{}

// NewUpgradeCmd creates a new upgrade command
func NewUpgradeCmd(plugins []plugin.Metadata) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(plugins, cobraCmd, args)
		},
	}

	return upgradeCmd
}

// Run executes the command logic
func (cmd *UpgradeCmd) Run(plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Execute plugin hook
	err := plugin.ExecutePluginHook(plugins, cobraCmd, args, "upgrade", "", "", nil)
	if err != nil {
		return err
	}

	err = upgrade.Upgrade()
	if err != nil {
		return errors.Errorf("Couldn't upgrade: %v", err)
	}

	return nil
}
