package update

import (
	"github.com/devspace-cloud/devspace/pkg/util/factory"
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
	f.GetLog().StartWait("Updating plugin " + args[0])
	defer f.GetLog().StopWait()

	err := f.NewPluginManager(f.GetLog()).Update(args[0], cmd.Version)
	if err != nil {
		return err
	}

	f.GetLog().StopWait()
	f.GetLog().Donef("Successfully updated plugin %s", args[0])
	return nil
}