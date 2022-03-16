package config

import "sync"

type RuntimeVariables interface {
	// ListRuntimeVariables returns the runtime variables
	ListRuntimeVariables() map[string]interface{}

	// GetRuntimeVariable retrieves a single runtime variable
	GetRuntimeVariable(key string) (interface{}, bool)

	// SetRuntimeVariable allows to set a runtime variable
	SetRuntimeVariable(key string, value interface{})
}

func newRuntimeVariables() RuntimeVariables {
	return &runtimeVariables{
		runtimeVariables: make(map[string]interface{}),
	}
}

type runtimeVariables struct {
	runtimeVariablesMutex sync.Mutex
	runtimeVariables      map[string]interface{}
}

func (c *runtimeVariables) GetRuntimeVariable(key string) (interface{}, bool) {
	c.runtimeVariablesMutex.Lock()
	defer c.runtimeVariablesMutex.Unlock()

	val, ok := c.runtimeVariables[key]
	return val, ok
}

func (c *runtimeVariables) SetRuntimeVariable(key string, value interface{}) {
	c.runtimeVariablesMutex.Lock()
	defer c.runtimeVariablesMutex.Unlock()

	c.runtimeVariables[key] = value
}

func (c *runtimeVariables) ListRuntimeVariables() map[string]interface{} {
	c.runtimeVariablesMutex.Lock()
	defer c.runtimeVariablesMutex.Unlock()

	retVars := map[string]interface{}{}
	for k, v := range c.runtimeVariables {
		retVars[k] = v
	}

	return retVars
}
