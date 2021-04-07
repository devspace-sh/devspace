package loader

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"strings"
)

type Parser interface {
	Parse(rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error)
}

func NewDefaultParser() Parser {
	return &defaultParser{}
}

type defaultParser struct{}

func (d *defaultParser) Parse(rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	// delete the commands, since we don't need it in a normal scenario
	delete(rawConfig, "commands")

	return fillVariablesAndParse(rawConfig, vars, resolver, options, log)
}

func NewWithCommandsParser() Parser {
	return &withCommandsParser{}
}

type withCommandsParser struct{}

func (d *withCommandsParser) Parse(rawConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	return fillVariablesAndParse(rawConfig, vars, resolver, options, log)
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

	return fillVariablesAndParse(preparedConfig, vars, resolver, options, log)
}

func fillVariablesAndParse(preparedConfig map[interface{}]interface{}, vars []*latest.Variable, resolver variable.Resolver, options *ConfigOptions, log log.Logger) (*latest.Config, error) {
	// fill in variables
	err := fillVariables(resolver, preparedConfig, vars, options)
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

// fillVariables fills in the given vars into the prepared config
func fillVariables(resolver variable.Resolver, preparedConfig map[interface{}]interface{}, vars []*latest.Variable, options *ConfigOptions) error {
	// Find out what vars are really used
	varsUsed, err := resolver.FindVariables(preparedConfig, vars)
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
			err = askQuestions(resolver, newVars)
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

func askQuestions(resolver variable.Resolver, vars []*latest.Variable) error {
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
