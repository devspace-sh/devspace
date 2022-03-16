package list

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"

	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
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
Lists all DevSpace configurations for this project
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
	configLoader, err := f.NewConfigLoader("")
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	config, err := configLoader.LoadWithParser(context.Background(), nil, nil, loader.NewProfilesParser(), nil, logger)
	if err != nil {
		return err
	}

	profiles := config.Config().Profiles

	// Specify the table column names
	headerColumnNames := []string{
		"Name",
		"Description",
	}

	configRows := make([][]string, 0, len(profiles))
	for _, profile := range profiles {
		configRows = append(configRows, []string{
			profile.Name,
			profile.Description,
		})
	}

	log.PrintTable(logger, headerColumnNames, configRows)
	return nil
}
