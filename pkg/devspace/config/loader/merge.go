package loader

import (
	"encoding/json"
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// ApplyStrategicMerge applies the strategic merge patches
func ApplyStrategicMerge(config map[interface{}]interface{}, profile map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	if profile == nil || profile["strategicMerge"] == nil {
		return config, nil
	}

	mergeMap, ok := profile["strategicMerge"].(map[interface{}]interface{})
	if !ok {
		return nil, errors.Errorf("profiles.%v.strategicMerge is not an object", profile["name"])
	}

	mergeBytes, err := json.Marshal(convertFrom(mergeMap))
	if err != nil {
		return nil, errors.Wrap(err, "marshal merge")
	}

	originalBytes, err := json.Marshal(convertFrom(config))
	if err != nil {
		return nil, errors.Wrap(err, "marshal merge")
	}

	out, err := strategicpatch.StrategicMergePatch(originalBytes, mergeBytes, &latest.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "create strategic merge patch")
	}

	strMap := map[string]interface{}{}
	err = json.Unmarshal(out, &strMap)
	if err != nil {
		return nil, err
	}

	return convertBack(strMap).(map[interface{}]interface{}), nil
}

// ApplyMerge applies the merge patches
func ApplyMerge(config map[interface{}]interface{}, profile map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	if profile == nil || profile["merge"] == nil {
		return config, nil
	}

	mergeMap, ok := profile["merge"].(map[interface{}]interface{})
	if !ok {
		return nil, errors.Errorf("profiles.%v.merge is not an object", profile["name"])
	}

	mergeBytes, err := json.Marshal(convertFrom(mergeMap))
	if err != nil {
		return nil, errors.Wrap(err, "marshal merge")
	}

	originalBytes, err := json.Marshal(convertFrom(config))
	if err != nil {
		return nil, errors.Wrap(err, "marshal merge")
	}

	out, err := jsonpatch.MergePatch(originalBytes, mergeBytes)
	if err != nil {
		return nil, errors.Wrap(err, "create merge patch")
	}

	strMap := map[string]interface{}{}
	err = json.Unmarshal(out, &strMap)
	if err != nil {
		return nil, err
	}

	return convertBack(strMap).(map[interface{}]interface{}), nil
}

func convertBack(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		m := map[interface{}]interface{}{}
		for k, v2 := range x {
			m[k] = convertBack(v2)
		}
		v = m

	case []interface{}:
		for i, v2 := range x {
			x[i] = convertBack(v2)
		}

	case map[interface{}]interface{}:
		for k, v2 := range x {
			x[k] = convertBack(v2)
		}
	}

	return v
}

func convertFrom(v interface{}) interface{} {
	switch x := v.(type) {
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v2 := range x {
			switch k2 := k.(type) {
			case string: // Fast check if it's already a string
				m[k2] = convertFrom(v2)
			default:
				m[fmt.Sprint(k)] = convertFrom(v2)
			}
		}
		v = m

	case []interface{}:
		for i, v2 := range x {
			x[i] = convertFrom(v2)
		}

	case map[string]interface{}:
		for k, v2 := range x {
			x[k] = convertFrom(v2)
		}
	}

	return v
}
