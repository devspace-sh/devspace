package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

type Parser interface {
	Parse(rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error)
}

func NewDefaultParser() Parser {
	return &defaultParser{}
}

type defaultParser struct{}

func (d *defaultParser) Parse(rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	// modify the config
	delete(rawConfig, "commands")

	// fill in variables
	err := fillVariables(resolver, rawConfig, vars, options)
	if err != nil {
		return nil, err
	}

	// Now convert the whole config to latest
	latestConfig, err := versions.Parse(rawConfig, log)
	if err != nil {
		return nil, errors.Wrap(err, "convert config")
	}

	return latestConfig, nil
}

func NewCommandsParser() Parser {
	return &commandsParser{}
}

type commandsParser struct {
}

func (c *commandsParser) Parse(rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	// modify the config
	preparedConfig, err := versions.ParseCommands(rawConfig)
	if err != nil {
		return nil, err
	}

	// fill in variables
	err = fillVariables(resolver, preparedConfig, vars, options)
	if err != nil {
		return nil, err
	}

	// Now convert the whole config to latest
	latestConfig, err := versions.Parse(preparedConfig, log)
	if err != nil {
		return nil, errors.Wrap(err, "convert config")
	}

	return latestConfig, nil
}
