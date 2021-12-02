package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/expression"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"path/filepath"
)

type Parser interface {
	Parse(configPath string, originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error)
}

func NewDefaultParser() Parser {
	return &defaultParser{}
}

type defaultParser struct{}

func (d *defaultParser) Parse(configPath string, originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	// delete the commands, since we don't need it in a normal scenario
	delete(rawConfig, "commands")

	return fillVariablesAndParse(configPath, rawConfig, vars, resolver, options, log)
}

func NewWithCommandsParser() Parser {
	return &withCommandsParser{}
}

type withCommandsParser struct{}

func (d *withCommandsParser) Parse(configPath string, originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	return fillVariablesAndParse(configPath, rawConfig, vars, resolver, options, log)
}

func NewCommandsParser() Parser {
	return &commandsParser{}
}

type commandsParser struct{}

func (c *commandsParser) Parse(configPath string, originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	// modify the config
	preparedConfig, err := versions.ParseCommands(rawConfig)
	if err != nil {
		return nil, err
	}

	return fillVariablesAndParse(configPath, preparedConfig, vars, resolver, options, log)
}

func NewProfilesParser() Parser {
	return &profilesParser{}
}

type profilesParser struct{}

func (p *profilesParser) Parse(configPath string, originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	rawMap, err := copyRaw(originalRawConfig)
	if err != nil {
		return nil, err
	}

	profiles, ok := rawMap["profiles"].([]interface{})
	if !ok {
		profiles = []interface{}{}
	}

	retProfiles := []*latest.ProfileConfig{}
	for _, profile := range profiles {
		profileMap, ok := profile.(map[interface{}]interface{})
		if !ok {
			continue
		}

		profileConfig := &latest.ProfileConfig{}
		o, err := yaml.Marshal(profileMap)
		if err != nil {
			continue
		}
		err = yaml.Unmarshal(o, profileConfig)
		if err != nil {
			continue
		}

		retProfiles = append(retProfiles, profileConfig)
	}

	retConfig := latest.NewRaw()
	retConfig.Profiles = retProfiles
	return retConfig, nil
}

func fillVariablesAndParse(configPath string, preparedConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	// fill in variables
	preparedConfigInterface, err := resolver.FindAndFillVariables(preparedConfig, vars)
	if err != nil {
		return nil, err
	}

	// execute expressions
	preparedConfigInterface, err = expression.ResolveAllExpressions(preparedConfigInterface, filepath.Dir(configPath))
	if err != nil {
		return nil, err
	}

	// fill in variables again
	preparedConfigInterface, err = resolver.FindAndFillVariables(preparedConfigInterface, vars)
	if err != nil {
		return nil, err
	}

	// Now convert the whole config to latest
	latestConfig, err := versions.Parse(preparedConfigInterface.(map[interface{}]interface{}), log)
	if err != nil {
		return nil, errors.Wrap(err, "convert config")
	}

	return latestConfig, nil
}
