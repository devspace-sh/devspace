package remove

import (
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

type pluginCmd struct {
}

func newPluginCmd(f factory.Factory) *cobra.Command {
	cmd := &pluginCmd{}
	pluginCmd := &cobra.Command{
		Use:   "plugin",
		Short: "Removes a devspace plugin",
		Long: `
#######################################################
############# devspace remove plugin ##################
#######################################################
Removes a plugin

devspace remove plugin my-plugin 
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, cobraCmd, args)
		}}

	return pluginCmd
}

// Run executes the command logic
func (cmd *pluginCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	pluginManager := f.NewPluginManager(f.GetLog())
	_, oldPlugin, err := pluginManager.GetByName(args[0])
	if err != nil {
		return err
	} else if oldPlugin != nil {
		// Execute plugin hook
		err = plugin.ExecutePluginHookAt(*oldPlugin, "before:removePlugin", "before_remove")
		if err != nil {
			return err
		}
	}

	f.GetLog().Info("Removing plugin " + args[0] + "...")
	err = pluginManager.Remove(args[0])
	if err != nil {
		return err
	}

	f.GetLog().Donef("Successfully removed plugin %s", args[0])
	return nil
}
