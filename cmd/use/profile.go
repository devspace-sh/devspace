package use

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"

	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ProfileCmd struct {
	Reset bool
}

func newProfileCmd() *cobra.Command {
	cmd := &ProfileCmd{}

	useProfile := &cobra.Command{
		Use:   "profile",
		Short: "Use a specific DevSpace profile",
		Long: `
#######################################################
################ devspace use profile #################
#######################################################
Use a specific DevSpace profile

Example:
devspace use profile production
devspace use profile staging
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
func (cmd *ProfileCmd) RunUseProfile(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := logpkg.GetInstance()
	configLoader := loader.NewConfigLoader(nil, logpkg.Discard)
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

	profileName := ""
	if cmd.Reset == false {
		if len(args) > 0 {
			profileName = args[0]
		} else {
			profileName, err = log.Question(&survey.QuestionOptions{
				Question: "Please select a profile to use",
				Options:  profiles,
			})
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
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Exchange active config
	generatedConfig.ActiveProfile = profileName

	// Save generated config
	err = configLoader.SaveGenerated(generatedConfig)
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
