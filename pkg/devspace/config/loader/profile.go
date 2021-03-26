package loader

import (
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch/v5"
	yamlpatch "github.com/krishicks/yaml-patch"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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

// ApplyReplace applies the replaces
func ApplyReplace(config map[interface{}]interface{}, profile map[interface{}]interface{}) error {
	if profile == nil || profile["replace"] == nil {
		return nil
	}

	replaceMap, ok := profile["replace"].(map[interface{}]interface{})
	if !ok {
		return errors.Errorf("profiles.%v.replace is not an object", profile["name"])
	}

	for k, v := range replaceMap {
		config[k] = v
	}

	return nil
}

// ApplyPatches applies the patches to the config if defined
func ApplyPatches(data map[interface{}]interface{}, profile map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	out, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}

	patchesRaw, ok := profile["patches"]
	if !ok {
		return data, nil
	}

	patchesArr, ok := patchesRaw.([]interface{})
	if !ok {
		return nil, errors.Errorf("profile.%v.patches is not an array", profile["name"])
	} else if len(patchesArr) == 0 {
		return data, nil
	}

	configPatches := []*latest.PatchConfig{}
	err = util.Convert(patchesArr, &configPatches)
	if err != nil {
		return nil, errors.Wrap(err, "convert patches")
	}

	patches := yamlpatch.Patch{}
	for idx, patch := range configPatches {
		if patch.Operation == "" {
			return nil, errors.Errorf("profiles.%v.patches.%d.op is missing", profile["name"], idx)
		} else if patch.Path == "" {
			return nil, errors.Errorf("profiles.%v.patches.%d.path is missing", profile["name"], idx)
		}

		newPatch := yamlpatch.Operation{
			Op:   yamlpatch.Op(patch.Operation),
			Path: yamlpatch.OpPath(transformPath(patch.Path)),
			From: yamlpatch.OpPath(transformPath(patch.From)),
		}

		if patch.Value != nil {
			newPatch.Value = yamlpatch.NewNode(&patch.Value)
		}

		if string(newPatch.Op) == "add" && patch.Path[0] != '/' {
			// In yamlpath the user has to add a '/-' to append to an array which is often confusing
			// if the '/-' is not added the operation is essentially an replace. So what we do here is check
			// if the operation is add, the path points to an array and the specified path was not XPath -> then we will just append the /-
			target, _ := findPath(&newPatch.Path, data)
			if _, ok := target.([]interface{}); ok {
				newPatch.Path = yamlpatch.OpPath(strings.TrimSuffix(string(newPatch.Path), "/-") + "/-")
			}
		}

		patches = append(patches, newPatch)
	}

	out, err = patches.Apply(out)
	if err != nil {
		return nil, errors.Wrap(err, "apply patches")
	}

	newConfig := map[interface{}]interface{}{}
	err = yaml.Unmarshal(out, &newConfig)
	if err != nil {
		return nil, err
	}

	return newConfig, nil
}

var replaceArrayRegEx = regexp.MustCompile("\\[\\\"?([^\\]\\\"]+)\\\"?\\]")

func findPath(path *yamlpatch.OpPath, c interface{}) (interface{}, error) {
	container := yamlpatch.NewNode(&c).Container()
	if path.ContainsExtendedSyntax() {
		paths := yamlpatch.NewPathFinder(container).Find(path.String())
		if paths == nil {
			return nil, fmt.Errorf("could not expand pointer: %s", path.String())
		}

		for _, p := range paths {
			op := yamlpatch.OpPath(p)
			return findPath(&op, c)
		}

		return nil, nil
	}

	parts, key, err := path.Decompose()
	if err != nil {
		return nil, err
	}

	parts = append(parts, key)
	foundContainer := c
	for _, part := range parts {
		// If map
		iMap, ok := foundContainer.(map[interface{}]interface{})
		if ok {
			foundContainer, ok = iMap[part]
			if ok == false {
				return nil, errors.Errorf("cannot find key %s in object", part)
			}

			continue
		}

		iArray, ok := foundContainer.([]interface{})
		if ok {
			i, err := strconv.Atoi(part)
			if err != nil {
				return nil, err
			}

			if i >= 0 && i <= len(iArray)-1 {
				foundContainer = iArray[i]
				continue
			}

			return nil, errors.Errorf("unable to access invalid index: %d", i)
		}

		return nil, errors.Errorf("cannot access part %s because value is not an object or array", part)
	}

	return foundContainer, nil
}

func transformPath(path string) string {
	// Test if XPath
	if path == "" || path[0] == '/' {
		return path
	}

	// Replace t[0] -> t/0
	path = replaceArrayRegEx.ReplaceAllString(path, "/$1")
	path = strings.Replace(path, ".", "/", -1)
	path = "/" + path

	return path
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
