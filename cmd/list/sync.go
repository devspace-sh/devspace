package list

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
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
		Run:  cmd.RunListSync,
	}

	return syncCmd
}

// RunListSync runs the list sync command logic
func (cmd *syncCmd) RunListSync(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	config := configutil.GetConfig(cmd.KubeContext, cmd.Profile)

	if config.Dev.Sync == nil || len(config.Dev.Sync) == 0 {
		log.Info("No sync paths are configured. Run `devspace add sync` to add new sync path\n")
		return
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
}
