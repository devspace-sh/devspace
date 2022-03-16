package loader

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/patch"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/pkg/errors"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// ApplyStrategicMerge applies the strategic merge patches
func ApplyStrategicMerge(config map[string]interface{}, profile map[string]interface{}) (map[string]interface{}, error) {
	if profile == nil || profile["strategicMerge"] == nil {
		return config, nil
	}

	mergeMap, ok := profile["strategicMerge"].(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("profiles.%v.strategicMerge is not an object", profile["name"])
	}

	mergeBytes, err := json.Marshal(mergeMap)
	if err != nil {
		return nil, errors.Wrap(err, "marshal merge")
	}

	originalBytes, err := json.Marshal(config)
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

	return strMap, nil
}

// ApplyMerge applies the merge patches
func ApplyMerge(config map[string]interface{}, profile map[string]interface{}) (map[string]interface{}, error) {
	if profile == nil || profile["merge"] == nil {
		return config, nil
	}

	mergeMap, ok := profile["merge"].(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("profiles.%v.merge is not an object", profile["name"])
	}

	mergeBytes, err := json.Marshal(mergeMap)
	if err != nil {
		return nil, errors.Wrap(err, "marshal merge")
	}

	originalBytes, err := json.Marshal(config)
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

	return strMap, nil
}

// ApplyReplace applies the replaces
func ApplyReplace(config map[string]interface{}, profile map[string]interface{}) error {
	if profile == nil || profile["replace"] == nil {
		return nil
	}

	replaceMap, ok := profile["replace"].(map[string]interface{})
	if !ok {
		return errors.Errorf("profiles.%v.replace is not an object", profile["name"])
	}

	for k, v := range replaceMap {
		config[k] = v
	}

	return nil
}

// ApplyPatches applies the patches to the config if defined
func ApplyPatches(data map[string]interface{}, profile map[string]interface{}) (map[string]interface{}, error) {
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
	err := util.Convert(patchesArr, &configPatches)
	if err != nil {
		return nil, errors.Wrap(err, "convert patches")
	}

	return ApplyPatchesOnObject(data, configPatches)
}

func ApplyPatchesOnObject(data map[string]interface{}, configPatches []*latest.PatchConfig) (map[string]interface{}, error) {
	out, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(out, &doc); err != nil {
		return nil, err
	}

	patches := patch.Patch{}
	for idx, patchConfig := range configPatches {
		if patchConfig.Operation == "" {
			return nil, errors.Errorf("patches.%d.op is missing", idx)
		} else if patchConfig.Path == "" {
			return nil, errors.Errorf("patches.%d.path is missing", idx)
		}

		newPatch := patch.Operation{
			Op:   patch.Op(patchConfig.Operation),
			Path: patch.OpPath(transformPath(patchConfig.Path)),
		}

		if patchConfig.Value != nil {
			value, err := patch.NewNode(&patchConfig.Value)
			if err != nil {
				return nil, errors.Errorf("patches.%d.value is invalid", idx)
			}
			newPatch.Value = value
		}

		if string(newPatch.Op) == "remove" && patchConfig.Path[0] != '/' {
			// figure out automatically if the path to remove is not there and just skip the patch
			target, _ := findPath(&newPatch.Path, &doc)
			if target == nil {
				continue
			}
		}

		if string(newPatch.Op) == "replace" && patchConfig.Path[0] != '/' {
			// figure out automatically if to use add or replace based on if the target path exists or not
			target, _ := findPath(&newPatch.Path, &doc)
			if target == nil {
				newPatch.Op = patch.Op("add")
			}
		}

		patches = append(patches, newPatch)
	}

	out, err = patches.Apply(out)
	if err != nil {
		return nil, errors.Wrap(err, "apply patches")
	}

	newConfig := map[string]interface{}{}
	err = yaml.Unmarshal(out, &newConfig)
	if err != nil {
		return nil, err
	}

	return newConfig, nil
}

func findPath(path *patch.OpPath, doc *yaml.Node) (interface{}, error) {
	pathFinder, err := yamlpath.NewPath(string(*path))
	if err != nil {
		return nil, err
	}

	matches, err := pathFinder.Find(doc)
	if err != nil {
		return nil, err
	}

	if len(matches) > 0 {
		return matches[0], nil
	}

	return nil, nil
}

var legacyExtendedSyntaxRegEx = regexp.MustCompile(`(?i)([^\=]+)=([^\.\=\>\<\~]+)`)
var hasFilterRegEx = regexp.MustCompile(`(?i)\[\?.*\)\]`)
var indexXPathRegEx = regexp.MustCompile(`\/(\d+|\*)\/`)
var trailingIndexXPathRegEx = regexp.MustCompile(`\/(\d+|\*)$`)
var rootXPathRegEx = regexp.MustCompile(`^\/`)
var numeric = regexp.MustCompile(`^\d+$`)

func transformPath(path string) string {
	if path == "" {
		return path
	}

	rewrittenPath := path

	if legacyExtendedSyntaxRegEx.MatchString(path) {
		// Using property=value selectors
		rewriteTokens := []string{}
		tokens := strings.Split(path, ".")
		for _, token := range tokens {
			rewriteToken := token
			if legacyExtendedSyntaxRegEx.MatchString(token) {
				filterTokens := legacyExtendedSyntaxRegEx.FindStringSubmatch(token)
				if numeric.MatchString((filterTokens[2])) {
					rewriteToken = fmt.Sprintf("[?(@.%s=='%s' || @.%s==%s)]", filterTokens[1], filterTokens[2], filterTokens[1], filterTokens[2])
				} else {
					rewriteToken = fmt.Sprintf("[?(@.%s=='%s')]", filterTokens[1], filterTokens[2])
				}
			}
			rewriteTokens = append(rewriteTokens, rewriteToken)
		}
		rewrittenPath = strings.Join(rewriteTokens, ".")
		rewrittenPath = strings.ReplaceAll(rewrittenPath, ".[?", "[?")
	} else if strings.Contains(path, "/") && !hasFilterRegEx.MatchString(path) {
		// Is XPath
		rewrittenPath = indexXPathRegEx.ReplaceAllString(path, "[$1].")
		rewrittenPath = trailingIndexXPathRegEx.ReplaceAllString(rewrittenPath, "[$1]")
		rewrittenPath = rootXPathRegEx.ReplaceAllLiteralString(rewrittenPath, "$.")
	}

	return rewrittenPath
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
