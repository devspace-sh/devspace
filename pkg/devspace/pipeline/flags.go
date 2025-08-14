package pipeline

import (
	"fmt"
	"strconv"

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
			switch pipelineFlag.Default.(type) {
			case float64:
				floatVal, ok := pipelineFlag.Default.(float64)
				if !ok {
					return nil, fmt.Errorf("default is not an integer")
				}
				return int(floatVal), nil
			case int:
				intVal, ok := pipelineFlag.Default.(int)
				if !ok {
					return nil, fmt.Errorf("default is not an integer")
				}
				return intVal, nil
			case string:
				strVal, ok := pipelineFlag.Default.(string)
				if !ok {
					return nil, fmt.Errorf("default is not an integer")
				}
				intVal, err := strconv.ParseInt(strVal, 10, 0)
				if err != nil {
					return nil, err
				}
				return int(intVal), nil
			}
			return nil, fmt.Errorf("default is not an integer")
		}
		return val, nil
	case latest.PipelineFlagTypeStringArray:
		val := []string{}
		if pipelineFlag.Default != nil {
			for _, anyVal := range pipelineFlag.Default.([]interface{}) {
				strVal, ok := anyVal.(string)
				if !ok {
					return nil, fmt.Errorf("default is not a string array")
				}
				val = append(val, strVal)
			}
		}
		return val, nil
	}

	return nil, fmt.Errorf("unsupported flag type")
}
