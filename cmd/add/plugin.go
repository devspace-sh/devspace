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
		Short: "Add a plugin to devspace",
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
			return cmd.Run(f, cobraCmd, args)
		}}

	pluginCmd.Flags().StringVar(&cmd.Version, "version", "", "The git tag to use")
	return pluginCmd
}

// Run executes the command logic
func (cmd *pluginCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	f.GetLog().StartWait("Installing plugin " + args[0])
	defer f.GetLog().StopWait()

	addedPlugin, err := f.NewPluginManager(f.GetLog()).Add(args[0], cmd.Version)
	if err != nil {
		return err
	}

	f.GetLog().StopWait()
	f.GetLog().Donef("Successfully installed plugin %s", args[0])

	// Execute plugin hook
	err = plugin.ExecutePluginHook([]plugin.Metadata{*addedPlugin}, cobraCmd, args, "after_install", "", "", nil)
	if err != nil {
		return err
	}

	return nil
}
