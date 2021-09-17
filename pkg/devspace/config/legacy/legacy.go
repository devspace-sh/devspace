package legacy

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
)

// LegacyInteractiveMode mutates the config if interactive mode should be used. This will be removed in future
func LegacyInteractiveMode(config *latest.Config, forceInteractiveMode bool, forceTerminal bool, log log.Logger) (bool, error) {
	// adjust config for interactive mode
	interactiveModeEnabled := forceInteractiveMode || config.Dev.InteractiveEnabled
	if interactiveModeEnabled {
		if config.Dev.InteractiveEnabled {
			log.Warn("You are using a deprecated config option dev.interactive.defaultEnabled, please upgrade your config to " + latest.Version + " and use dev.terminal instead")
		}

		images := config.Images
		if config.Dev.InteractiveImages == nil && config.Dev.Terminal == nil {
			if config.Images == nil || len(config.Images) == 0 {
				return false, errors.New(message.ConfigNoImages)
			}

			imageNames := make([]string, 0, len(images))
			for k := range images {
				imageNames = append(imageNames, k)
			}

			// If only one image exists, use it, otherwise show image picker
			var err error
			imageName := ""
			if len(imageNames) == 1 {
				imageName = imageNames[0]
			} else {
				question := "Which image do you want to build using the 'ENTRYPOINT [sleep, 999999]' override?"
				imageName, err = log.Question(&survey.QuestionOptions{
					Question: question,
					Options:  imageNames,
				})
				if err != nil {
					return false, err
				}
			}

			config.Dev.InteractiveImages = []*latest.InteractiveImageConfig{
				{
					Name: imageName,
				},
			}
		}

		// set image entrypoints if necessary
		for _, imageConf := range config.Dev.InteractiveImages {
			if forceTerminal {
				imageConf.Entrypoint = nil
				imageConf.Cmd = nil
			} else if imageConf.Entrypoint == nil && imageConf.Cmd == nil {
				imageConf.Entrypoint = []string{"sleep"}
				imageConf.Cmd = []string{"999999999"}
			}

			for imageConfName, imageOverrideConfig := range config.Images {
				if imageConf.Name == imageConfName {
					if imageConf.Entrypoint != nil {
						imageOverrideConfig.Entrypoint = imageConf.Entrypoint
					}
					if imageConf.Cmd != nil {
						imageOverrideConfig.Cmd = imageConf.Cmd
					}
					break
				}
			}

			if imageConf.Entrypoint != nil && imageConf.Cmd != nil {
				log.Infof("Override image '%s' entrypoint with %+v and cmd with %+v", ansi.Color(imageConf.Name, "white+b"), imageConf.Entrypoint, imageConf.Cmd)
			} else if imageConf.Entrypoint != nil {
				log.Infof("Override image '%s' entrypoint with %+v", ansi.Color(imageConf.Name, "white+b"), imageConf.Entrypoint)
			} else if imageConf.Cmd != nil {
				log.Infof("Override image '%s' cmd with %+v", ansi.Color(imageConf.Name, "white+b"), imageConf.Cmd)
			}
		}

		// make sure terminal is enabled
		if config.Dev.Terminal == nil {
			config.Dev.Terminal = &latest.Terminal{}
			if len(config.Dev.InteractiveImages) > 0 {
				config.Dev.Terminal.ImageSelector = fmt.Sprintf("image(%s):tag(%s)", config.Dev.InteractiveImages[0].Name, config.Dev.InteractiveImages[0].Name)
			}
		} else if config.Dev.Terminal.Disabled {
			config.Dev.Terminal.Disabled = false
		}

		// make sure this is never used again
		config.Dev.InteractiveEnabled = false
		config.Dev.InteractiveImages = nil
	}

	return interactiveModeEnabled, nil
}
