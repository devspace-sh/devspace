package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/util/log"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/pkg/errors"
)

func (l *configLoader) applyProfiles(data map[interface{}]interface{}, options *ConfigOptions, log log.Logger) (map[interface{}]interface{}, error) {
	// Get profile
	profiles, err := versions.ParseProfile(filepath.Dir(l.configPath), data, options.Profile, options.ProfileParents, options.ProfileRefresh, log)
	if err != nil {
		return nil, err
	}

	// Now delete not needed parts from config
	delete(data, "profiles")

	// Apply profiles
	for i := len(profiles) - 1; i >= 0; i-- {
		// Apply replace
		err = ApplyReplace(data, profiles[i])
		if err != nil {
			return nil, err
		}

		// Apply merge
		data, err = ApplyMerge(data, profiles[i])
		if err != nil {
			return nil, err
		}

		// Apply strategic merge
		data, err = ApplyStrategicMerge(data, profiles[i])
		if err != nil {
			return nil, err
		}

		// Apply patches
		data, err = ApplyPatches(data, profiles[i])
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

// parseConfig fills the variables in the data and parses the config
func (l *configLoader) parseConfig(resolver variable.Resolver, data map[interface{}]interface{}, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	// apply the profiles
	data, err := l.applyProfiles(data, options, log)
	if err != nil {
		return nil, err
	}

	// delete the commands section
	delete(data, "commands")

	// Load defined variables
	vars, err := versions.ParseVariables(data, log)
	if err != nil {
		return nil, err
	}

	// Delete vars from config
	delete(data, "vars")

	// Fill in variables
	err = l.fillVariables(resolver, data, vars, options)
	if err != nil {
		return nil, err
	}

	// Now convert the whole config to latest
	latestConfig, err := versions.Parse(data, log)
	if err != nil {
		return nil, errors.Wrap(err, "convert config")
	}

	return latestConfig, nil
}

// fillVariables fills in the given vars into the prepared config
func (l *configLoader) fillVariables(resolver variable.Resolver, preparedConfig map[interface{}]interface{}, vars []*latest.Variable, options *ConfigOptions) error {
	// Find out what vars are really used
	varsUsed, err := resolver.FindVariables(preparedConfig)
	if err != nil {
		return err
	}

	// parse cli --var's, the resolver will cache them for us
	_, err = resolver.ConvertFlags(options.Vars)
	if err != nil {
		return err
	}

	// Fill used defined variables
	if len(vars) > 0 {
		newVars := []*latest.Variable{}
		for _, v := range vars {
			if varsUsed[strings.TrimSpace(v.Name)] {
				newVars = append(newVars, v)
			}
		}

		if len(newVars) > 0 {
			err = l.askQuestions(resolver, newVars)
			if err != nil {
				return err
			}
		}
	}

	// Walk over data and fill in variables
	err = resolver.FillVariables(preparedConfig)
	if err != nil {
		return err
	}

	return nil
}

func (l *configLoader) askQuestions(resolver variable.Resolver, vars []*latest.Variable) error {
	for _, definition := range vars {
		name := strings.TrimSpace(definition.Name)

		// fill the variable with definition
		_, err := resolver.Resolve(name, definition)
		if err != nil {
			return err
		}
	}

	return nil
}
