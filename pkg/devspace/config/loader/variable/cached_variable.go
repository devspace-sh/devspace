package variable

import "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

func NewCachedValueVariable(value interface{}) Variable {
	return &cachedValueVariable{
		value: value,
	}
}

type cachedValueVariable struct {
	name  string
	value interface{}
}

func (c *cachedValueVariable) Load(definition *latest.Variable) (interface{}, error) {
	return c.value, nil
}
