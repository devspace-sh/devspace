package pipeline

import (
	"fmt"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func GetDefaultValue(pipelineFlag latest.PipelineFlag) (interface{}, error) {
	var ok bool

	switch pipelineFlag.Type {
	case "", latest.PipelineFlagTypeBoolean:
		val := false
		if pipelineFlag.Default != nil {
			val, ok = pipelineFlag.Default.(bool)
			if !ok {
				return nil, fmt.Errorf(" default is not a boolean")
			}
		}
		return val, nil
	case latest.PipelineFlagTypeString:
		val := ""
		if pipelineFlag.Default != nil {
			val, ok = pipelineFlag.Default.(string)
			if !ok {
				return nil, fmt.Errorf("default is not a string")
			}
		}
		return val, nil
	case latest.PipelineFlagTypeInteger:
		val := 0
		if pipelineFlag.Default != nil {
			val, ok = pipelineFlag.Default.(int)
			if !ok {
				return nil, fmt.Errorf("default is not an integer")
			}
		}
		return val, nil
	case latest.PipelineFlagTypeStringArray:
		val := []string{}
		if pipelineFlag.Default != nil {
			val, ok = pipelineFlag.Default.([]string)
			if !ok {
				return nil, fmt.Errorf("default is not a string array")
			}
		}
		return val, nil
	}

	return nil, fmt.Errorf("unsupported flag type")
}
