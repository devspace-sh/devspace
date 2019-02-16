package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// ConvertCmd holds the login cmd flags
type ConvertCmd struct{}

// NewConvertCmd creates a new login command
func NewConvertCmd() *cobra.Command {
	cmd := &ConvertCmd{}

	convertCmd := &cobra.Command{
		Use:   "convert",
		Short: "Converts the active config to the current config version",
		Long: `
	#######################################################
	################### devspace login ####################
	#######################################################
	Converts the active config to the current config version

	Note: convert does not upgrade the overwrite configs
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunConvert,
	}

	return convertCmd
}

// RunConvert executes the functionality devspace convert
func (cmd *ConvertCmd) RunConvert(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	// Get config
	configutil.GetConfigWithoutDefaults(false)

	// Save it
	err = configutil.SaveBaseConfig()
	if err != nil {
		log.Fatalf("Error saving config: %v", err)
	}

	log.Infof("Successfully converted base config to current version")
}
