package configutil

import (
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

func convertDotDevSpaceConfigToDevSpaceYaml(basePath string) error {
	oldConfigPath := filepath.Join(basePath, ".devspace", "config.yaml")
	newConfigPath := filepath.Join(basePath, DefaultConfigPath)

	// Convert old to new config.yaml
	_, err := os.Stat(newConfigPath)
	if os.IsNotExist(err) {
		// Check if .devspace/config.yaml exists
		_, err := os.Stat(oldConfigPath)
		if os.IsNotExist(err) == false {
			err := os.Rename(oldConfigPath, newConfigPath)
			if err != nil {
				return errors.Wrap(err, "rename")
			}

			log.Infof("Renamed old config %s to new config %s", oldConfigPath, newConfigPath)
		}
	}

	oldConfigsPath := filepath.Join(basePath, ".devspace", "configs.yaml")
	newConfigsPath := filepath.Join(basePath, DefaultConfigsPath)

	// Convert old to new configs.yaml
	_, err = os.Stat(newConfigsPath)
	if os.IsNotExist(err) {
		// Check if .devspace/configs.yaml exists
		_, err = os.Stat(oldConfigsPath)
		if os.IsNotExist(err) == false {
			err = os.Rename(oldConfigsPath, newConfigsPath)
			if err != nil {
				return errors.Wrap(err, "rename")
			}

			log.Infof("Renamed old configs %s to new configs %s", oldConfigsPath, newConfigsPath)
		}
	}

	oldVarsPath := filepath.Join(basePath, ".devspace", "vars.yaml")
	newVarsPath := filepath.Join(basePath, DefaultVarsPath)

	// Convert old to new configs.yaml
	_, err = os.Stat(newVarsPath)
	if os.IsNotExist(err) {
		// Check if .devspace/vars.yaml exists
		_, err = os.Stat(oldVarsPath)
		if os.IsNotExist(err) == false {
			err = os.Rename(oldVarsPath, newVarsPath)
			if err != nil {
				return errors.Wrap(err, "rename")
			}

			log.Infof("Renamed old vars %s to new vars %s", oldVarsPath, newVarsPath)
		}
	}

	return nil
}
