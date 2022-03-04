package loader

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Parser interface {
	Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error)
}

func NewDefaultParser() Parser {
	return &defaultParser{}
}

type defaultParser struct{}

func (d *defaultParser) Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	// delete the commands, since we don't need it in a normal scenario
	delete(rawConfig, "commands")

	return fillVariablesAndParse(ctx, resolver, rawConfig, log)
}

func NewWithCommandsParser() Parser {
	return &withCommandsParser{}
}

type withCommandsParser struct{}

func (d *withCommandsParser) Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	return fillVariablesAndParse(ctx, resolver, rawConfig, log)
}

func NewCommandsParser() Parser {
	return &commandsParser{}
}

type commandsParser struct{}

func (c *commandsParser) Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	// modify the config
	preparedConfig, err := versions.GetCommands(rawConfig)
	if err != nil {
		return nil, nil, err
	}

	return fillVariablesAndParse(ctx, resolver, preparedConfig, log)
}

func NewProfilesParser() Parser {
	return &profilesParser{}
}

type profilesParser struct{}

func (p *profilesParser) Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	rawMap, err := copyRaw(originalRawConfig)
	if err != nil {
		return nil, nil, err
	}

	profiles, ok := rawMap["profiles"].([]interface{})
	if !ok {
		profiles = []interface{}{}
	}

	retProfiles := []*latest.ProfileConfig{}
	for _, profile := range profiles {
		profileMap, ok := profile.(map[string]interface{})
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
	return retConfig, rawMap, nil
}

func fillVariablesAndParse(ctx context.Context, resolver variable.Resolver, preparedConfig map[string]interface{}, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	// fill in variables and expressions (leave out
	preparedConfigInterface, err := resolver.FillVariablesExclude(ctx, preparedConfig, runtime.Locations)
	if err != nil {
		return nil, nil, err
	}

	latestConfig, err := Convert(preparedConfigInterface.(map[string]interface{}), log)
	if err != nil {
		return nil, nil, err
	}

	return latestConfig, preparedConfigInterface.(map[string]interface{}), nil
}

func Convert(prepatedConfig map[string]interface{}, log log.Logger) (*latest.Config, error) {
	// Now convert the whole config to latest
	latestConfig, err := versions.Parse(prepatedConfig, log)
	if err != nil {
		return nil, errors.Wrap(err, "convert config")
	}

	// now we validate the config
	err = Validate(latestConfig)
	if err != nil {
		return nil, err
	}

	return latestConfig, nil
}
