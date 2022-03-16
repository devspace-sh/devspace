package variable

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/pkg/errors"
)

// NewNoneVariable creates a new variable from source type none
func NewNoneVariable(name string) Variable {
	return &noneVariable{
		name: name,
	}
}

type noneVariable struct {
	name string
}

func (n *noneVariable) Load(ctx context.Context, definition *latest.Variable) (interface{}, error) {
	if definition.Default == nil {
		return nil, errors.Errorf("couldn't set variable '%s', because source is '%s' but the default value is empty", n.name, latest.VariableSourceNone)
	}

	return definition.Default, nil
}
