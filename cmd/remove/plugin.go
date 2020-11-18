package remove

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
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
		err = plugin.ExecutePluginHook([]plugin.Metadata{*oldPlugin}, cobraCmd, args, "before_remove", "", "", nil)
		if err != nil {
			return err
		}
	}

	f.GetLog().StartWait("Removing plugin " + args[0])
	defer f.GetLog().StopWait()

	err = pluginManager.Remove(args[0])
	if err != nil {
		return err
	}

	f.GetLog().StopWait()
	f.GetLog().Donef("Successfully removed plugin %s", args[0])
	return nil
}
