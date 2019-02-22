package list

import (
	"os"
	"strconv"

	"github.com/covexo/devspace/pkg/devspace/config/configs"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type configsCmd struct{}

func newConfigsCmd() *cobra.Command {
	cmd := &configsCmd{}

	configsCmd := &cobra.Command{
		Use:   "configs",
		Short: "Lists the defined configurations",
		Long: `
	#######################################################
	############## devspace list configs ##################
	#######################################################
	Lists the defined devspace configuartions
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
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	// Check if configs.yaml exists
	_, err = os.Stat(configutil.DefaultConfigsPath)
	if err != nil {
		log.Info("Please create a .devspace/configs.yaml to specify multiple configurations")
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
		if config.Overwrites != nil {
			overrides = len(*config.Overwrites)
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
