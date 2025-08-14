package loader

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"gopkg.in/yaml.v3"
)

type Parser interface {
	Parse(
		ctx context.Context,
		originalRawConfig map[string]interface{},
		rawConfig map[string]interface{},
		resolver variable.Resolver,
		log log.Logger,
	) (*latest.Config, map[string]interface{}, error)
}

func NewDefaultParser() Parser {
	return &defaultParser{}
}

type defaultParser struct{}

func (d *defaultParser) Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	// delete the commands, since we don't need it in a normal scenario
	return fillVariablesExcludeAndParse(ctx, resolver, rawConfig, log)
}

func NewCommandsPipelinesParser() Parser {
	return &commandsPipelinesParser{}
}

type commandsPipelinesParser struct{}

func (c *commandsPipelinesParser) Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	// modify the config
	preparedConfig, err := versions.Get(rawConfig, "commands", "pipelines")
	if err != nil {
		return nil, nil, err
	}

	return fillVariablesExcludeAndParse(ctx, resolver, preparedConfig, log)
}

func NewCommandsParser() Parser {
	return &commandsParser{}
}

type commandsParser struct{}

func (c *commandsParser) Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	// modify the config
	preparedConfig, err := versions.Get(rawConfig, "commands")
	if err != nil {
		return nil, nil, err
	}

	return fillVariablesExcludeAndParse(ctx, resolver, preparedConfig, log)
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
		err = yamlutil.Unmarshal(o, profileConfig)
		if err != nil {
			continue
		}

		retProfiles = append(retProfiles, profileConfig)
	}

	retConfig := latest.NewRaw()
	retConfig.Profiles = retProfiles
	return retConfig, rawMap, nil
}

func NewEagerParser() Parser {
	return &eagerParser{}
}

type eagerParser struct{}

func (e *eagerParser) Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	return fillAllVariablesAndParse(ctx, resolver, rawConfig, log)
}

func fillAllVariablesAndParse(ctx context.Context, resolver variable.Resolver, preparedConfig map[string]interface{}, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	return fillVariablesAndParse(ctx, resolver, preparedConfig, log)
}

func fillVariablesExcludeAndParse(ctx context.Context, resolver variable.Resolver, preparedConfig map[string]interface{}, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	return fillVariablesAndParse(ctx, resolver, preparedConfig, log, runtime.Locations...)
}

func fillVariablesAndParse(ctx context.Context, resolver variable.Resolver, preparedConfig map[string]interface{}, log log.Logger, excludedPaths ...string) (*latest.Config, map[string]interface{}, error) {
	// fill in variables and expressions
	preparedConfigInterface, err := resolver.FillVariablesExclude(ctx, preparedConfig, false, excludedPaths)
	if err != nil {
		return nil, nil, err
	}

	latestConfig, err := versions.Parse(preparedConfigInterface.(map[string]interface{}), log)
	if err != nil {
		return nil, nil, err
	}

	return latestConfig, preparedConfigInterface.(map[string]interface{}), nil
}

func EachDevContainer(devPod *latest.DevPod, each func(devContainer *latest.DevContainer) bool) {
	if len(devPod.Containers) > 0 {
		for _, devContainer := range devPod.Containers {
			cont := each(devContainer)
			if !cont {
				break
			}
		}
	} else {
		_ = each(&devPod.DevContainer)
	}
}
