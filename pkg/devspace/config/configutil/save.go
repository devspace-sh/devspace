package configutil

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/covexo/devspace/pkg/util/fsutil"
	yaml "gopkg.in/yaml.v2"
)

// SaveConfig writes the data of a config to its yaml file
func SaveConfig() error {
	// Don't save custom config files
	if ConfigPath != DefaultConfigPath || OverwriteConfigPath != DefaultOverwriteConfigPath {
		return nil
	}

	// default and overwrite values
	configToIgnore := makeConfig()

	Merge(&configToIgnore, defaultConfig, true)
	Merge(&configToIgnore, overwriteConfig, true)

	// generates config without default and overwrite values
	configMapRaw, _, err := splitConfigs(config, configRaw, configToIgnore)

	// generates overwriteConfig
	_, overwriteMapRaw, err := splitConfigs(config, configRaw, overwriteConfig)

	if err != nil {
		return err
	}

	configMap, _ := configMapRaw.(map[interface{}]interface{})
	overwriteMap, _ := overwriteMapRaw.(map[interface{}]interface{})

	configYaml, err := yaml.Marshal(configMap)
	if err != nil {
		return err
	}

	configDir := filepath.Dir(ConfigPath)
	os.MkdirAll(configDir, os.ModePerm)

	// Check if .gitignore exists
	_, err = os.Stat(filepath.Join(configDir, ".gitignore"))
	if os.IsNotExist(err) {
		fsutil.WriteToFile([]byte(configGitignore), filepath.Join(configDir, ".gitignore"))
	}

	writeErr := ioutil.WriteFile(ConfigPath, configYaml, os.ModePerm)
	if writeErr != nil {
		return writeErr
	}

	if overwriteMap != nil {
		overwriteConfigYaml, err := yaml.Marshal(overwriteMap)
		if err != nil {
			return err
		}

		return ioutil.WriteFile(OverwriteConfigPath, overwriteConfigYaml, os.ModePerm)
	}

	return nil
}
