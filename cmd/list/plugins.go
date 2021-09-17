package list

import (
	"strconv"

	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type pluginsCmd struct {
}

func newPluginsCmd(f factory.Factory) *cobra.Command {
	cmd := &pluginsCmd{}
	pluginCmd := &cobra.Command{
		Use:   "plugins",
		Short: "Lists all installed devspace plugins",
		Long: `
#######################################################
############# devspace list plugins ##################
#######################################################
Lists all installed plugins

devspace list plugins
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		}}

	return pluginCmd
}

// Run executes the command logic
func (cmd *pluginsCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	plugins, err := f.NewPluginManager(f.GetLog()).List()
	if err != nil {
		return err
	}

	headerColumnNames := []string{
		"Name",
		"Version",
		"Commands",
		"Vars",
	}

	// Transform values into string arrays
	rows := make([][]string, 0, len(plugins))
	for _, plugin := range plugins {
		row := []string{
			plugin.Name,
			plugin.Version,
			strconv.Itoa(len(plugin.Commands)),
			strconv.Itoa(len(plugin.Vars)),
		}

		rows = append(rows, row)
	}

	log.PrintTable(f.GetLog(), headerColumnNames, rows)
	return nil
}
