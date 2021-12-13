package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Parser interface {
	Parse(originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, error)
}

func NewDefaultParser() Parser {
	return &defaultParser{}
}

type defaultParser struct{}

func (d *defaultParser) Parse(originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, error) {
	// delete the commands, since we don't need it in a normal scenario
	delete(rawConfig, "commands")

	return fillVariablesAndParse(resolver, rawConfig, log)
}

func NewWithCommandsParser() Parser {
	return &withCommandsParser{}
}

type withCommandsParser struct{}

func (d *withCommandsParser) Parse(originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, error) {
	return fillVariablesAndParse(resolver, rawConfig, log)
}

func NewCommandsParser() Parser {
	return &commandsParser{}
}

type commandsParser struct{}

func (c *commandsParser) Parse(originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, error) {
	// modify the config
	preparedConfig, err := versions.ParseCommands(rawConfig)
	if err != nil {
		return nil, err
	}

	return fillVariablesAndParse(resolver, preparedConfig, log)
}

func NewProfilesParser() Parser {
	return &profilesParser{}
}

type profilesParser struct{}

func (p *profilesParser) Parse(originalRawConfig map[interface{}]interface{}, rawConfig map[interface{}]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, error) {
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

func fillVariablesAndParse(resolver variable.Resolver, preparedConfig map[interface{}]interface{}, log log.Logger) (*latest.Config, error) {
	// fill in variables and expressions (leave out
	preparedConfigInterface, err := resolver.FillVariablesExclude(preparedConfig, runtime.Locations)
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
