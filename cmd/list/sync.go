package list

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type syncCmd struct {
	*flags.GlobalFlags
}

func newSyncCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: cmd.RunListSync,
	}

	return syncCmd
}

// RunListSync runs the list sync command logic
func (cmd *syncCmd) RunListSync(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log.GetInstance())
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	config, err := configLoader.Load()
	if err != nil {
		return err
	}

	if config.Dev.Sync == nil || len(config.Dev.Sync) == 0 {
		log.GetInstance().Info("No sync paths are configured. Run `devspace add sync` to add new sync path\n")
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

	log.PrintTable(log.GetInstance(), headerColumnNames, syncPaths)
	return nil
}
