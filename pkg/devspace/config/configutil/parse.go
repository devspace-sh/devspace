package configutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	varspkg "github.com/devspace-cloud/devspace/pkg/util/vars"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// LoadedVars holds all variables that were loaded
var LoadedVars = make(map[string]string)

func varMatchFn(path, key, value string) bool {
	return varspkg.VarMatchRegex.MatchString(value)
}

// GetProfiles retrieves all available profiles
func GetProfiles(basePath string) ([]string, error) {
	bytes, err := ioutil.ReadFile(filepath.Join(basePath, constants.DefaultConfigPath))
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
func ParseCommands(generatedConfig *generated.Config, data map[interface{}]interface{}, options *ConfigOptions, log log.Logger) ([]*latest.CommandConfig, error) {
	if options == nil {
		options = &ConfigOptions{}
	}

	// Load defined variables
	vars, err := versions.ParseVariables(data)
	if err != nil {
		return nil, err
	}

	// Parse commands
	config, err := versions.ParseCommands(data)
	if err != nil {
		return nil, err
	}

	preparedConfig := map[interface{}]interface{}{}
	err = util.Convert(config, &preparedConfig)
	if err != nil {
		return nil, err
	}

	// Fill in variables
	err = FillVariables(generatedConfig, preparedConfig, vars, options, log)
	if err != nil {
		return nil, err
	}

	// Now parse the whole config
	parsedConfig, err := versions.Parse(preparedConfig, options.LoadedVars)
	if err != nil {
		return nil, errors.Wrap(err, "parse config")
	}

	return parsedConfig.Commands, nil
}

// ParseConfig fills the variables in the data and parses the config
func ParseConfig(generatedConfig *generated.Config, data map[interface{}]interface{}, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	// Load defined variables
	vars, err := versions.ParseVariables(data)
	if err != nil {
		return nil, err
	}

	// Prepare config for variable loading
	preparedConfig, err := versions.ParseProfile(data, options.Profile)
	if err != nil {
		return nil, err
	}

	// Fill in variables
	err = FillVariables(generatedConfig, preparedConfig, vars, options, log)
	if err != nil {
		return nil, err
	}

	// Now parse the whole config
	parsedConfig, err := versions.Parse(preparedConfig, options.LoadedVars)
	if err != nil {
		return nil, errors.Wrap(err, "parse config")
	}

	return parsedConfig, nil
}

// FillVariables fills in the given vars into the prepared config
func FillVariables(generatedConfig *generated.Config, preparedConfig map[interface{}]interface{}, vars []*latest.Variable, options *ConfigOptions, log log.Logger) error {
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
	cmdVars, err := parseVarsFromOptions(options)
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
			err = askQuestions(generatedConfig, newVars, cmdVars, log)
			if err != nil {
				return err
			}
		}
	}

	// Fill predefined vars
	err = fillPredefinedVars(options)
	if err != nil {
		return err
	}

	// Walk over data and fill in variables
	err = walk.Walk(preparedConfig, varMatchFn, func(path, value string) (interface{}, error) {
		return varReplaceFn(path, value, generatedConfig, cmdVars, options, log)
	})
	if err != nil {
		return err
	}

	return nil
}

func parseVarsFromOptions(options *ConfigOptions) (map[string]string, error) {
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

func askQuestions(generatedConfig *generated.Config, vars []*latest.Variable, cmdVars map[string]string, log log.Logger) error {
	for _, variable := range vars {
		name := strings.TrimSpace(variable.Name)

		// Check if var is provided through cli
		if _, ok := cmdVars[name]; ok {
			continue
		}

		isInEnv := os.Getenv(name) != ""
		// Check if variable is defined to be env var (source: env) but not defined
		if variable.Source != nil && *variable.Source == latest.VariableSourceEnv && isInEnv == false {
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

		generatedConfig.Vars[name], err = askQuestion(variable, log)
		if err != nil {
			return err
		}
	}

	return nil
}

func varReplaceFn(path, value string, generatedConfig *generated.Config, cmdVars map[string]string, options *ConfigOptions, log log.Logger) (interface{}, error) {
	// Save old value
	if options.LoadedVars != nil {
		options.LoadedVars[path] = value
	}

	return varspkg.ParseString(value, func(v string) (string, error) { return resolveVar(v, generatedConfig, cmdVars, options, log) })
}

func resolveVar(varName string, generatedConfig *generated.Config, cmdVars map[string]string, options *ConfigOptions, log log.Logger) (string, error) {
	// Is cli variable?
	if val, ok := cmdVars[varName]; ok {
		return val, nil
	}

	// Is predefined variable?
	found, value, err := getPredefinedVar(varName, generatedConfig, options)
	if err != nil {
		return "", err
	} else if found {
		return value, nil
	}

	// Is in generated config?
	if _, ok := generatedConfig.Vars[varName]; ok {
		return generatedConfig.Vars[varName], nil
	}

	// Is in environment?
	if os.Getenv(varName) != "" {
		return os.Getenv(varName), nil
	}

	// Ask for variable
	generatedConfig.Vars[varName], err = askQuestion(&latest.Variable{
		Question: "Please enter a value for " + varName,
	}, log)
	if err != nil {
		return "", err
	}

	return generatedConfig.Vars[varName], nil
}

func askQuestion(variable *latest.Variable, log log.Logger) (string, error) {
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

	answer, err := survey.Question(params, log)
	if err != nil {
		return "", err
	}

	return answer, nil
}
