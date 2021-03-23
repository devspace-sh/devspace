package update

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
		Short: "Updates a devspace plugin",
		Long: `
#######################################################
############# devspace update plugin ##################
#######################################################
Updates a plugin

devspace update plugin my-plugin 
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
	pluginManager := f.NewPluginManager(f.GetLog())
	_, oldPlugin, err := pluginManager.GetByName(args[0])
	if err != nil {
		return err
	} else if oldPlugin != nil {
		// Execute plugin hook
		err = plugin.ExecutePluginHook([]plugin.Metadata{*oldPlugin}, cobraCmd, args, "before_update", "", "", nil)
		if err != nil {
			return err
		}
	}

	f.GetLog().StartWait("Updating plugin " + args[0])
	defer f.GetLog().StopWait()

	updatedPlugin, err := pluginManager.Update(args[0], cmd.Version)
	if err != nil {
		if newestVersion, ok := err.(*plugin.NewestVersionError); ok {
			f.GetLog().Info(newestVersion.Error())
			return nil
		}

		return err
	}

	f.GetLog().StopWait()
	f.GetLog().Donef("Successfully updated plugin %s", args[0])

	// Execute plugin hook
	err = plugin.ExecutePluginHook([]plugin.Metadata{*updatedPlugin}, cobraCmd, args, "after_update", "", "", nil)
	if err != nil {
		return err
	}

	return nil
}
