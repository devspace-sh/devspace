package list

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type syncCmd struct {
	*flags.GlobalFlags
}

func newSyncCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &syncCmd{GlobalFlags: globalFlags}

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Lists sync configuration",
		Long: `
#######################################################
################# devspace list sync ##################
#######################################################
Lists the sync configuration
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListSync(f, cobraCmd, args)
		}}

	return syncCmd
}

// RunListSync runs the list sync command logic
func (cmd *syncCmd) RunListSync(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	// Set config root
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	configInterface, err := configLoader.Load(cmd.ToConfigOptions(), logger)
	if err != nil {
		return err
	}

	config := configInterface.Config()
	if config.Dev.Sync == nil || len(config.Dev.Sync) == 0 {
		logger.Info("No sync paths are configured. Run `devspace add sync` to add new sync path\n")
		return nil
	}

	headerColumnNames := []string{
		"Label Selector",
		"Local Path",
		"Container Path",
		"Excluded Paths",
	}

	syncPaths := make([][]string, 0, len(config.Dev.Sync))

	// Transform values into string arrays
	for _, value := range config.Dev.Sync {
		selector := ""

		for k, v := range value.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}

			selector += k + "=" + v
		}
		excludedPaths := ""

		if value.ExcludePaths != nil {
			for _, v := range value.ExcludePaths {
				if len(excludedPaths) > 0 {
					excludedPaths += ", "
				}

				excludedPaths += v
			}
		}

		syncPaths = append(syncPaths, []string{
			selector,
			value.LocalSubPath,
			value.ContainerPath,
			excludedPaths,
		})
	}

	log.PrintTable(logger, headerColumnNames, syncPaths)
	return nil
}
