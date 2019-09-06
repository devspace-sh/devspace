package configutil

import "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"

// ApplyReplace applies the replaces
func ApplyReplace(config *latest.Config) error {
	if len(config.Profiles) != 1 {
		return nil
	}

	loadedProfile := config.Profiles[0]
	if loadedProfile.Replace == nil {
		return nil
	}

	if loadedProfile.Replace.Images != nil {
		config.Images = loadedProfile.Replace.Images
	}
	if loadedProfile.Replace.Deployments != nil {
		config.Deployments = loadedProfile.Replace.Deployments
	}
	if loadedProfile.Replace.Dev != nil {
		config.Dev = loadedProfile.Replace.Dev
	}
	if loadedProfile.Replace.Dependencies != nil {
		config.Dependencies = loadedProfile.Replace.Dependencies
	}
	if loadedProfile.Replace.Hooks != nil {
		config.Hooks = loadedProfile.Replace.Hooks
	}

	return nil
}
