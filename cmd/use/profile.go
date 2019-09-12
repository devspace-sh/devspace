package use

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type profileCmd struct {
	Reset bool
}

func newProfileCmd() *cobra.Command {
	cmd := &profileCmd{}

	useProfile := &cobra.Command{
		Use:   "profile",
		Short: "Use a specific DevSpace profile",
		Long: `
#######################################################
################ devspace use profile #################
#######################################################
Use a specific DevSpace profile

Example:
devspace use profile myconfig
devspace use profile --reset
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		RunE: cmd.RunUseProfile,
	}

	useProfile.Flags().BoolVar(&cmd.Reset, "reset", false, "Don't use a profile anymore")

	return useProfile
}

// RunUseProfile executes the "devspace use config command" logic
func (cmd *profileCmd) RunUseProfile(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	profiles, err := configutil.GetProfiles(".")
	if err != nil {
		return err
	}

	profileName := ""
	if cmd.Reset == false {
		if len(args) > 0 {
			profileName = args[0]
		} else {
			profileName, err = survey.Question(&survey.QuestionOptions{
				Question: "Please select a profile to use",
				Options:  profiles,
			}, log.GetInstance())
			if err != nil {
				return err
			}
		}

		// Check if config exists
		found := false
		for _, profile := range profiles {
			if profile == profileName {
				found = true
				break
			}
		}

		if found == false {
			return errors.Errorf("Profile '%s' does not exist in devspace.yaml", profileName)
		}
	}

	// Load generated config
	generatedConfig, err := generated.LoadConfig("")
	if err != nil {
		return err
	}

	// Exchange active config
	generatedConfig.ActiveProfile = profileName

	// Save generated config
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		return err
	}

	if cmd.Reset {
		log.Info("Successfully resetted profile")
	} else {
		log.Infof("Successfully switched to profile '%s'", profileName)
	}

	return nil
}
