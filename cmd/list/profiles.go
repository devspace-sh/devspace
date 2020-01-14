package list

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type profilesCmd struct{}

func newProfilesCmd(f factory.Factory) *cobra.Command {
	cmd := &profilesCmd{}

	profilesCmd := &cobra.Command{
		Use:   "profiles",
		Short: "Lists all DevSpace profiles",
		Long: `
#######################################################
############## devspace list profiles #################
#######################################################
Lists all DevSpace configuartions for this project
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListProfiles(f, cobraCmd, args)
		}}

	return profilesCmd
}

// RunListProfiles runs the list profiles command logic
func (cmd *profilesCmd) RunListProfiles(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	// Set config root
	configLoader := f.NewConfigLoader(nil, logger)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	profiles, err := loader.GetProfiles(".")
	if err != nil {
		return err
	}

	// Load generated config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Specify the table column names
	headerColumnNames := []string{
		"Name",
		"Active",
	}

	configRows := make([][]string, 0, len(profiles))

	for _, profile := range profiles {
		configRows = append(configRows, []string{
			profile,
			strconv.FormatBool(profile == generatedConfig.ActiveProfile),
		})
	}

	log.PrintTable(logger, headerColumnNames, configRows)
	return nil
}
