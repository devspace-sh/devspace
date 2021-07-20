package variable

import (
	"os"
	"strconv"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// NewDefaultVariable creates a new variable for the sources default, all or input
func NewDefaultVariable(name string, cache map[string]string, log log.Logger) Variable {
	return &defaultVariable{
		name:  name,
		cache: cache,
		log:   log,
	}
}

type defaultVariable struct {
	name  string
	cache map[string]string
	log   log.Logger
}

func (d *defaultVariable) Load(definition *latest.Variable) (interface{}, error) {
	if definition.Command != "" || len(definition.Commands) > 0 {
		return NewCommandVariable(d.name).Load(definition)
	}

	// Check environment
	value := os.Getenv(d.name)

	// Did we find it in the environment variables?
	if definition.Source != latest.VariableSourceInput && value != "" {
		return valueByType(value, definition.Default)
	}

	// Is cached
	if value, ok := d.cache[d.name]; !definition.NoCache && ok {
		return valueByType(value, definition.Default)
	}

	// Now ask the question
	value, err := askQuestion(definition, d.log)
	if err != nil {
		return nil, err
	}

	if !definition.NoCache {
		d.cache[d.name] = value
	}
	return valueByType(value, definition.Default)
}

func valueByType(value string, defaultValue interface{}) (interface{}, error) {
	if defaultValue == nil {
		return convertStringValue(value), nil
	}

	switch defaultValue.(type) {
	case int:
		r, err := strconv.Atoi(value)
		return r, err
	case bool:
		r, err := strconv.ParseBool(value)
		return r, err
	default:
		return value, nil
	}
}
