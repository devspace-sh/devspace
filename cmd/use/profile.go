package use

import (
	"github.com/loft-sh/devspace/pkg/util/factory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/loft-sh/devspace/pkg/util/survey"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ProfileCmd struct {
	Reset bool
}

func newProfileCmd(f factory.Factory) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunUseProfile(f, cobraCmd, args)
		},
	}

	useProfile.Flags().BoolVar(&cmd.Reset, "reset", false, "Don't use a profile anymore")

	return useProfile
}

// RunUseProfile executes the "devspace use config command" logic
func (cmd *ProfileCmd) RunUseProfile(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configLoader := f.NewConfigLoader("")
	configExists, err := configLoader.SetDevSpaceRoot(logpkg.Discard)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	config, err := configLoader.Load(nil, logpkg.Discard)
	if err != nil {
		return err
	}

	profileObjects, err := config.Profiles()
	if err != nil {
		return err
	}

	profiles := []string{}
	for _, p := range profileObjects {
		profiles = append(profiles, p.Name)
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
	generatedConfig := config.Generated()

	// Exchange active config
	generatedConfig.ActiveProfile = profileName

	// Save generated config
	err = configLoader.SaveGenerated(generatedConfig)
	if err != nil {
		return err
	}

	if cmd.Reset {
		log.Info("Successfully reset profile")
	} else {
		log.Infof("Successfully switched to profile '%s'", profileName)
	}

	return nil
}
