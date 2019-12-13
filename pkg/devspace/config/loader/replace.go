package loader

import "github.com/pkg/errors"

// ApplyReplace applies the replaces
func ApplyReplace(config map[interface{}]interface{}, profile map[interface{}]interface{}) error {
	if profile == nil || profile["replace"] == nil {
		return nil
	}

	replaceMap, ok := profile["replace"].(map[interface{}]interface{})
	if !ok {
		return errors.Errorf("profiles.%v.replace is not an object", profile["name"])
	}

	for k, v := range replaceMap {
		config[k] = v
	}

	return nil
}
