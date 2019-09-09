package latest

import (
	"context"
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
)

// Variables returns only the variables from the config
func Variables(data map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	retMap := map[interface{}]interface{}{}
	err := util.Convert(data, &retMap)
	if err != nil {
		return nil, err
	}

	return map[interface{}]interface{}{
		"version": Version,
		"vars":    retMap["vars"],
	}, nil
}

// Prepare prepares the given config for variable loading
func Prepare(ctx context.Context, data map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	loaded := map[interface{}]interface{}{}
	err := util.Convert(data, &loaded)
	if err != nil {
		return nil, err
	}

	// Delete vars definition
	delete(loaded, "vars")

	config := ""
	if ctx.Value(constants.ProfileContextKey) != nil {
		config = ctx.Value(constants.ProfileContextKey).(string)
	}

	if config == "" {
		delete(loaded, "profiles")
		return loaded, nil
	}

	// Convert to array
	profiles, ok := loaded["profiles"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("Couldn't load profile '%s': no profiles found", config)
	}

	// Search for config
	for _, profileMap := range profiles {
		configMap, ok := profileMap.(map[interface{}]interface{})
		if ok && configMap["name"] == config {
			loaded["profiles"] = []interface{}{profileMap}
			return loaded, nil
		}
	}

	// Couldn't find config
	return nil, fmt.Errorf("Couldn't find profile '%s'", config)
}
