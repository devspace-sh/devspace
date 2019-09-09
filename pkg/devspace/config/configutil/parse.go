package configutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"
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
		return nil, fmt.Errorf("Error parsing devspace.yaml: %v", err)
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
func ParseConfig(ctx context.Context, generatedConfig *generated.Config, data map[interface{}]interface{}, profile string) (*latest.Config, error) {
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
			err = askQuestions(generatedConfig, newVars)
			if err != nil {
				return nil, err
			}
		}
	}

	// Fill predefined vars
	err = fillPredefinedVars(ctx)
	if err != nil {
		return nil, err
	}

	// Walk over data and fill in variables
	err = walk.Walk(preparedConfig, varMatchFn, func(path, value string) (interface{}, error) { return varReplaceFn(ctx, path, value, generatedConfig) })
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

func askQuestions(generatedConfig *generated.Config, vars []*latest.Variable) error {
	for _, variable := range vars {
		name := strings.TrimSpace(variable.Name)

		isInEnv := os.Getenv(VarEnvPrefix+strings.ToUpper(name)) != "" || os.Getenv(name) != ""
		if variable.Source != nil && *variable.Source == latest.VariableSourceEnv && isInEnv == false {
			return fmt.Errorf("Couldn't find environment variable %s, but is needed for loading the config", name)
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
		generatedConfig.Vars[name] = askQuestion(variable)
	}

	return nil
}

func varReplaceFn(ctx context.Context, path, value string, generatedConfig *generated.Config) (interface{}, error) {
	// Save old value
	LoadedVars[path] = value

	return varspkg.ParseString(value, func(v string) (string, error) { return resolveVar(ctx, v, generatedConfig) })
}

func resolveVar(ctx context.Context, varName string, generatedConfig *generated.Config) (string, error) {
	// Is predefined variable?
	found, value, err := getPredefinedVar(ctx, varName)
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
	generatedConfig.Vars[varName] = askQuestion(&latest.Variable{
		Question: "Please enter a value for " + varName,
	})

	return generatedConfig.Vars[varName], nil
}

func askQuestion(variable *latest.Variable) string {
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

	return survey.Question(params)
}
