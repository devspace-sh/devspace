package list

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type profilesCmd struct{}

func newProfilesCmd() *cobra.Command {
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
		RunE: cmd.RunListProfiles,
	}

	return profilesCmd
}

// RunListProfiles runs the list profiles command logic
func (cmd *profilesCmd) RunListProfiles(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(nil, log.GetInstance())
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

	log.PrintTable(log.GetInstance(), headerColumnNames, configRows)
	return nil
}
