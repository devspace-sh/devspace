package list

import (
	"context"
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
	configLoader, _ := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	configInterface, err := configLoader.Load(context.TODO(), nil, cmd.ToConfigOptions(), logger)
	if err != nil {
		return err
	}

	config := configInterface.Config()
	syncPaths := make([][]string, 0)

	for _, dev := range config.Dev {
		if dev.Sync == nil || len(dev.Sync) == 0 {
			logger.Info("No sync paths are configured.")
			return nil
		}
		selector := ""
		for k, v := range dev.LabelSelector {
			if len(selector) > 0 {
				selector += ", "
			}
			selector += k + "=" + v
		}
		// Transform values into string arrays
		for _, value := range dev.Sync {
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
				value.Path,
				excludedPaths,
			})
		}
	}

	headerColumnNames := []string{
		"Label Selector",
		"Path (Local:Container)",
		"Excluded Paths",
	}

	log.PrintTable(logger, headerColumnNames, syncPaths)
	return nil
}
