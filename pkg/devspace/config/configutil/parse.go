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
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	varspkg "github.com/devspace-cloud/devspace/pkg/util/vars"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// VarEnvPrefix is the prefix environment variables should have in order to use them
const VarEnvPrefix = "DEVSPACE_VAR_"

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

// ParseConfig fills the variables in the data and parses the config
func ParseConfig(generatedConfig *generated.Config, data map[interface{}]interface{}, kubeContext, profile string, log log.Logger) (*latest.Config, error) {
	// Load defined variables
	vars, err := versions.ParseVariables(data)
	if err != nil {
		return nil, err
	}

	// Prepare config for variable loading
	preparedConfig, err := versions.Prepare(data, profile)
	if err != nil {
		return nil, err
	}

	// Find out what vars are really used
	varsUsed := map[string]bool{}
	err = walk.Walk(preparedConfig, varMatchFn, func(path, value string) (interface{}, error) {
		return varspkg.ParseString(value, func(v string) (string, error) {
			varsUsed[v] = true
			return "${" + v + "}", nil
		})
	})
	if err != nil {
		return nil, err
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
			err = askQuestions(generatedConfig, newVars, log)
			if err != nil {
				return nil, err
			}
		}
	}

	// Fill predefined vars
	err = fillPredefinedVars(kubeContext)
	if err != nil {
		return nil, err
	}

	// Walk over data and fill in variables
	err = walk.Walk(preparedConfig, varMatchFn, func(path, value string) (interface{}, error) {
		return varReplaceFn(path, value, generatedConfig, kubeContext, log)
	})
	if err != nil {
		return nil, err
	}

	// Now parse the whole config
	parsedConfig, err := versions.Parse(preparedConfig)
	if err != nil {
		return nil, errors.Wrap(err, "parse config")
	}

	return parsedConfig, nil
}

func askQuestions(generatedConfig *generated.Config, vars []*latest.Variable, log log.Logger) error {
	for _, variable := range vars {
		name := strings.TrimSpace(variable.Name)

		isInEnv := os.Getenv(VarEnvPrefix+strings.ToUpper(name)) != "" || os.Getenv(name) != ""
		// Check if variable is defined to be env var (source: env) but not defined
		if variable.Source != nil && *variable.Source == latest.VariableSourceEnv && isInEnv == false {
			// Use default value for env variable if it is configured
			if variable.Default != "" {
				return os.Setenv(name, variable.Default)
			} else {
				return errors.Errorf("Couldn't find environment variable %s, but is needed for loading the config", name)
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

		generatedConfig.Vars[name], err = askQuestion(variable, log)
		if err != nil {
			return err
		}
	}

	return nil
}

func varReplaceFn(path, value string, generatedConfig *generated.Config, kubeContext string, log log.Logger) (interface{}, error) {
	// Save old value
	LoadedVars[path] = value

	return varspkg.ParseString(value, func(v string) (string, error) { return resolveVar(v, generatedConfig, kubeContext, log) })
}

func resolveVar(varName string, generatedConfig *generated.Config, kubeContext string, log log.Logger) (string, error) {
	// Is predefined variable?
	found, value, err := getPredefinedVar(varName, kubeContext)
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
	if os.Getenv(VarEnvPrefix+strings.ToUpper(varName)) != "" {
		return os.Getenv(VarEnvPrefix + strings.ToUpper(varName)), nil
	} else if os.Getenv(varName) != "" {
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
