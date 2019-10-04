package list

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"

	"io/ioutil"

	"github.com/spf13/cobra"

	yaml "gopkg.in/yaml.v2"
)

type commandsCmd struct {
	*flags.GlobalFlags
}

func newCommandsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &commandsCmd{GlobalFlags: globalFlags}

	commandsCmd := &cobra.Command{
		Use:   "commands",
		Short: "Lists all custom DevSpace commands",
		Long: `
#######################################################
############## devspace list commands #################
#######################################################
Lists all DevSpace custom commands defined in the 
devspace.yaml
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: cmd.RunListProfiles,
	}

	return commandsCmd
}

// RunListCommands runs the list  command logic
func (cmd *commandsCmd) RunListProfiles(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot(log.GetInstance())
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Load commands
	bytes, err := ioutil.ReadFile(constants.DefaultConfigPath)
	if err != nil {
		return err
	}
	rawMap := map[interface{}]interface{}{}
	err = yaml.Unmarshal(bytes, &rawMap)
	if err != nil {
		return err
	}

	// Load generated config
	generatedConfig, err := generated.LoadConfig("")
	if err != nil {
		return err
	}

	// Parse commands
	commands, err := configutil.ParseCommands(generatedConfig, rawMap, log.GetInstance())
	if err != nil {
		return err
	}

	// Save variables
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		return err
	}

	// Specify the table column names
	headerColumnNames := []string{
		"Name",
		"Command",
	}

	rows := [][]string{}
	for _, command := range commands {
		rows = append(rows, []string{
			command.Name,
			command.Command,
		})
	}

	log.PrintTable(log.GetInstance(), headerColumnNames, rows)
	return nil
}
