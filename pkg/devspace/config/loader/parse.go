package loader

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	varspkg "github.com/devspace-cloud/devspace/pkg/util/vars"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

func varMatchFn(path, key, value string) bool {
	return varspkg.VarMatchRegex.MatchString(value)
}

// GetProfiles retrieves all available profiles
func (l *configLoader) GetProfiles() ([]string, error) {
	path := l.ConfigPath()
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rawMap := map[interface{}]interface{}{}
	err = yaml.Unmarshal(bytes, &rawMap)
	if err != nil {
		return nil, errors.Errorf("Error parsing devspace.yaml: %v", err)
	}

	profiles, ok := rawMap["profiles"].([]interface{})
	if !ok {
		profiles = []interface{}{}
	}

	profileNames := []string{}
	for _, profile := range profiles {
		profileMap, ok := profile.(map[interface{}]interface{})
		if !ok {
			continue
		}

		name, ok := profileMap["name"].(string)
		if !ok {
			continue
		}

		profileNames = append(profileNames, name)
	}

	return profileNames, nil
}

// ParseCommands fills the variables in the data and parses the commands
func (l *configLoader) ParseCommands() ([]*latest.CommandConfig, error) {
	data, err := l.LoadRaw()
	if err != nil {
		return nil, err
	}

	// Load defined variables
	vars, err := versions.ParseVariables(data, l.log)
	if err != nil {
		return nil, err
	}

	// Parse commands
	preparedConfig, err := versions.ParseCommands(data)
	if err != nil {
		return nil, err
	}

	// Fill in variables
	err = l.FillVariables(preparedConfig, vars)
	if err != nil {
		return nil, err
	}

	// Now parse the whole config
	parsedConfig, err := versions.Parse(preparedConfig, l.options.LoadedVars, l.log)
	if err != nil {
		return nil, errors.Wrap(err, "parse config")
	}

	return parsedConfig.Commands, nil
}

// parseConfig fills the variables in the data and parses the config
func (l *configLoader) parseConfig(data map[interface{}]interface{}) (*latest.Config, error) {
	// Get profile
	profile, err := versions.ParseProfile(data, l.options.Profile)
	if err != nil {
		return nil, err
	}

	// Now delete not needed parts from config
	delete(data, "profiles")
	delete(data, "commands")

	// Apply profile
	if profile != nil {
		// Apply replace
		err = ApplyReplace(data, profile)
		if err != nil {
			return nil, err
		}

		// Apply patches
		data, err = ApplyPatches(data, profile)
		if err != nil {
			return nil, err
		}
	}

	// Load defined variables
	vars, err := versions.ParseVariables(data, l.log)
	if err != nil {
		return nil, err
	}

	// Delete vars from config
	delete(data, "vars")

	// Fill in variables
	err = l.FillVariables(data, vars)
	if err != nil {
		return nil, err
	}

	// Now convert the whole config to latest
	latestConfig, err := versions.Parse(data, l.options.LoadedVars, l.log)
	if err != nil {
		return nil, errors.Wrap(err, "convert config")
	}

	return latestConfig, nil
}

// FillVariables fills in the given vars into the prepared config
func (l *configLoader) FillVariables(preparedConfig map[interface{}]interface{}, vars []*latest.Variable) error {
	// Find out what vars are really used
	varsUsed := map[string]bool{}
	err := walk.Walk(preparedConfig, varMatchFn, func(path, value string) (interface{}, error) {
		varspkg.ParseString(value, func(v string) (string, error) {
			varsUsed[v] = true
			return "", nil
		})

		return value, nil
	})
	if err != nil {
		return err
	}

	// Parse cli --var's
	cmdVars, err := ParseVarsFromOptions(l.options)
	if err != nil {
		return err
	}

	// Fill used defined variables
	if len(vars) > 0 {
		newVars := []*latest.Variable{}
		for _, variable := range vars {
			if varsUsed[strings.TrimSpace(variable.Name)] {
				newVars = append(newVars, variable)
			}
		}

		if len(newVars) > 0 {
			err = l.askQuestions(newVars, cmdVars)
			if err != nil {
				return err
			}
		}
	}

	// Walk over data and fill in variables
	l.resolvedVars = map[string]string{}
	err = walk.Walk(preparedConfig, varMatchFn, func(path, value string) (interface{}, error) {
		return l.VarReplaceFn(path, value, cmdVars)
	})
	if err != nil {
		return err
	}

	return nil
}

// ParseVarsFromOptions returns the variables from the given options
func ParseVarsFromOptions(options *ConfigOptions) (map[string]string, error) {
	vars := map[string]string{}

	for _, cmdVar := range options.Vars {
		idx := strings.Index(cmdVar, "=")
		if idx == -1 {
			return nil, errors.Errorf("Wrong --var format: %s, expected 'key=val'", cmdVar)
		}

		vars[strings.TrimSpace(cmdVar[:idx])] = strings.TrimSpace(cmdVar[idx+1:])
	}

	return vars, nil
}

func (l *configLoader) askQuestions(vars []*latest.Variable, cmdVars map[string]string) error {
	generatedConfig, err := l.Generated()
	if err != nil {
		return err
	}

	for _, variable := range vars {
		name := strings.TrimSpace(variable.Name)

		// Check if var is provided through cli
		if _, ok := cmdVars[name]; ok {
			continue
		}

		isInEnv := os.Getenv(name) != ""
		// Check if variable is defined to be env var (source: env) but not defined
		if variable.Source != nil {
			// Environment variable
			if *variable.Source == latest.VariableSourceEnv && isInEnv == false {
				// Use default value for env variable if it is configured
				if variable.Default != "" {
					err := os.Setenv(name, variable.Default)
					if err != nil {
						return err
					}

					continue
				}

				return errors.Errorf("Couldn't find environment variable %s, but is needed for loading the config", name)
			}

			// Source none variable
			if *variable.Source == latest.VariableSourceNone {
				if variable.Default != "" {
					cmdVars[name] = variable.Default
					continue
				}

				return errors.Errorf("Couldn't set variable '%s', because source is '%s' but the default value is empty", name, latest.VariableSourceNone)
			}
		}

		// Check if variable is in environment
		if variable.Source == nil || *variable.Source != latest.VariableSourceInput {
			if isInEnv {
				continue
			}
		}

		// Is cached
		if _, ok := generatedConfig.Vars[name]; ok {
			continue
		}

		// Ask question
		var err error

		generatedConfig.Vars[name], err = l.askQuestion(variable)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *configLoader) VarReplaceFn(path, value string, cmdVars map[string]string) (interface{}, error) {
	// Save old value
	if l.options.LoadedVars != nil {
		l.options.LoadedVars[path] = value
	}

	return varspkg.ParseString(value, func(v string) (string, error) {
		val, err := l.ResolveVar(v, cmdVars)
		if err != nil {
			return "", err
		}

		l.resolvedVars[v] = val
		return val, nil
	})
}

func (l *configLoader) ResolveVar(varName string, cmdVars map[string]string) (string, error) {
	// Is cli variable?
	if val, ok := cmdVars[varName]; ok {
		return val, nil
	}

	// Is predefined variable?
	found, value, err := l.resolvePredefinedVar(varName)
	if err != nil {
		return "", err
	} else if found {
		return value, nil
	}

	// Is in generated config?
	generatedConfig, err := l.Generated()
	if err != nil {
		return "", err
	}

	if _, ok := generatedConfig.Vars[varName]; ok {
		return generatedConfig.Vars[varName], nil
	}

	// Is in environment?
	if os.Getenv(varName) != "" {
		return os.Getenv(varName), nil
	}

	// Ask for variable
	generatedConfig.Vars[varName], err = l.askQuestion(&latest.Variable{
		Question: "Please enter a value for " + varName,
	})
	if err != nil {
		return "", err
	}

	return generatedConfig.Vars[varName], nil
}

func (l *configLoader) askQuestion(variable *latest.Variable) (string, error) {
	params := &survey.QuestionOptions{}

	if variable == nil {
		params.Question = "Please enter a value"
	} else {
		if variable.Question == "" {
			if variable.Name == "" {
				variable.Name = "variable"
			}

			params.Question = "Please enter a value for " + variable.Name
		} else {
			params.Question = variable.Question
		}

		if variable.Password {
			params.IsPassword = true
		}

		if variable.Default != "" {
			params.DefaultValue = variable.Default
		}

		if len(variable.Options) > 0 {
			params.Options = variable.Options
		} else if variable.ValidationPattern != "" {
			params.ValidationRegexPattern = variable.ValidationPattern

			if variable.ValidationMessage != "" {
				params.ValidationMessage = variable.ValidationMessage
			}
		}
	}

	answer, err := l.log.Question(params)
	if err != nil {
		return "", err
	}

	return answer, nil
}
