package add

import (
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

type pluginCmd struct {
	Version string
}

func newPluginCmd(f factory.Factory) *cobra.Command {
	cmd := &pluginCmd{}
	pluginCmd := &cobra.Command{
		Use:   "plugin",
		Short: "Adds a plugin to devspace",
		Long: `
#######################################################
############### devspace add plugin ###################
#######################################################
Adds a new plugin to devspace

devspace add plugin https://github.com/my-plugin/plugin
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, args)
		}}

	pluginCmd.Flags().StringVar(&cmd.Version, "version", "", "The git tag to use")
	return pluginCmd
}

// Run executes the command logic
func (cmd *pluginCmd) Run(f factory.Factory, args []string) error {
	f.GetLog().Info("Installing plugin " + args[0])
	addedPlugin, err := f.NewPluginManager(f.GetLog()).Add(args[0], cmd.Version)
	if err != nil {
		return err
	}
	f.GetLog().Donef("Successfully installed plugin %s", args[0])

	// Execute plugin hook
	err = plugin.ExecutePluginHookAt(*addedPlugin, "after:installPlugin", "after_install")
	if err != nil {
		return err
	}

	return nil
}
