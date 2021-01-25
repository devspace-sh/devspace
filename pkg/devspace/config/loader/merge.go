package loader

import (
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"reflect"
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

	schema, err := strategicpatch.NewPatchMetaFromStruct(&latest.Config{})
	if err != nil {
		return nil, err
	}

	out, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(originalBytes, mergeBytes, PatchMetaFromStruct{PatchMetaFromStruct: schema})
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

type PatchMetaFromStruct struct {
	strategicpatch.PatchMetaFromStruct
}

func LookupPatchMetadataForMap(t reflect.Type) (
	elemType reflect.Type, patchStrategies []string, patchMergeKey string, e error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Map && t.Kind() != reflect.Interface {
		e = fmt.Errorf("merging an object in json but data type is not map, instead is: %s",
			t.Kind().String())
		return
	}
	if t.Kind() == reflect.Interface {
		return t, []string{}, "", nil
	}

	return t.Elem(), []string{}, "", nil
}

// we have to override map handling since otherwise it would produce errors
func (s PatchMetaFromStruct) LookupPatchMetadataForSlice(key string) (strategicpatch.LookupPatchMeta, strategicpatch.PatchMeta, error) {
	subschema, patchMeta, err := s.LookupPatchMetadataForStruct(key)
	if err != nil {
		return nil, strategicpatch.PatchMeta{}, err
	}
	elemPatchMetaFromStruct := subschema.(PatchMetaFromStruct)
	t := elemPatchMetaFromStruct.T

	var elemType reflect.Type
	switch t.Kind() {
	// If t is an array or a slice, get the element type.
	// If element is still an array or a slice, return an error.
	// Otherwise, return element type.
	case reflect.Array, reflect.Slice:
		elemType = t.Elem()
		if elemType.Kind() == reflect.Array || elemType.Kind() == reflect.Slice {
			return nil, strategicpatch.PatchMeta{}, errors.New("unexpected slice of slice")
		}
	// If t is an pointer, get the underlying element.
	// If the underlying element is neither an array nor a slice, the pointer is pointing to a slice,
	// e.g. https://github.com/kubernetes/kubernetes/blob/bc22e206c79282487ea0bf5696d5ccec7e839a76/staging/src/k8s.io/apimachinery/pkg/util/strategicpatch/patch_test.go#L2782-L2822
	// If the underlying element is either an array or a slice, return its element type.
	case reflect.Ptr:
		t = t.Elem()
		if t.Kind() == reflect.Array || t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
			t = t.Elem()
		}
		elemType = t
	case reflect.Map:
		elemType = t.Elem()
	case reflect.Interface:
		elemType = t
	default:
		return nil, strategicpatch.PatchMeta{}, fmt.Errorf("expected slice or array type, but got: %s", t.Kind().String())
	}

	return PatchMetaFromStruct{strategicpatch.PatchMetaFromStruct{T: elemType}}, patchMeta, nil
}

// we have to override map handling since otherwise it would produce errors
func (s PatchMetaFromStruct) LookupPatchMetadataForStruct(key string) (strategicpatch.LookupPatchMeta, strategicpatch.PatchMeta, error) {
	fieldType, fieldPatchStrategies, fieldPatchMergeKey, err := LookupPatchMetadataForMap(s.PatchMetaFromStruct.T)
	if err != nil {
		l, p, err := s.PatchMetaFromStruct.LookupPatchMetadataForStruct(key)
		if err != nil {
			return nil, strategicpatch.PatchMeta{}, err
		}

		return PatchMetaFromStruct{l.(strategicpatch.PatchMetaFromStruct)}, p, err
	}

	patchMeta := strategicpatch.PatchMeta{}
	patchMeta.SetPatchMergeKey(fieldPatchMergeKey)
	patchMeta.SetPatchStrategies(fieldPatchStrategies)
	return PatchMetaFromStruct{strategicpatch.PatchMetaFromStruct{T: fieldType}},
		patchMeta, nil
}
