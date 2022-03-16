package variable

import (
	"context"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/pkg/errors"
)

// NewEnvVariable creates a new variable that is loaded without definition
func NewEnvVariable(name string) Variable {
	return &envVariable{
		name: name,
	}
}

type envVariable struct {
	name string
}

func (e *envVariable) Load(ctx context.Context, definition *latest.Variable) (interface{}, error) {
	// Check environment
	value := os.Getenv(e.name)

	// Use default value for env variable if it is configured
	if value == "" {
		if definition.Default == nil {
			return nil, errors.Errorf("couldn't find environment variable %s, but is needed for loading the config", e.name)
		}

		return definition.Default, nil
	}

	return convertStringValue(value), nil
}
