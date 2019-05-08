package list

import (
	"os"
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type configsCmd struct{}

func newConfigsCmd() *cobra.Command {
	cmd := &configsCmd{}

	configsCmd := &cobra.Command{
		Use:   "configs",
		Short: "Lists all DevSpace configurations",
		Long: `
#######################################################
############## devspace list configs ##################
#######################################################
Lists all DevSpace configuartions for this project
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListConfigs,
	}

	return configsCmd
}

// RunListConfigs runs the list configs command logic
func (cmd *configsCmd) RunListConfigs(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Check if configs.yaml exists
	_, err = os.Stat(configutil.DefaultConfigsPath)
	if err != nil {
		log.Infof("Please create a '%s' to specify multiple configurations", configutil.DefaultConfigsPath)
		return
	}

	configs := configs.Configs{}

	// Get configs
	err = configutil.LoadConfigs(&configs, configutil.DefaultConfigsPath)
	if err != nil {
		log.Fatalf("Error loading %s: %v", configutil.DefaultConfigsPath, err)
	}

	// Load generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Specify the table column names
	headerColumnNames := []string{
		"Name",
		"Active",
		"Path",
		"Vars",
		"Overwrites",
	}

	configRows := make([][]string, 0, len(configs))

	for configName, config := range configs {
		path := ""
		if config.Config.Path != nil {
			path = *config.Config.Path
		}

		overrides := 0
		if config.Overrides != nil {
			overrides = len(*config.Overrides)
		}

		configRows = append(configRows, []string{
			configName,
			strconv.FormatBool(configName == generatedConfig.ActiveConfig),
			path,
			strconv.FormatBool(config.Vars != nil),
			strconv.Itoa(overrides),
		})
	}

	log.PrintTable(headerColumnNames, configRows)
}
