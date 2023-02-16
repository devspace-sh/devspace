package loader

import (
	"encoding/json"
	"fmt"
	"reflect"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/patch"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/pkg/errors"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// ApplyMerge applies the merge patches
func ApplyMerge(config map[string]interface{}, profile *latest.ProfileConfig) (map[string]interface{}, error) {
	if profile == nil || profile.Merge == nil {
		return config, nil
	}

	var err error
	if profile.Merge.Hooks != nil {
		config, err = applyMerge(config, "hooks", *profile.Merge.Hooks)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.Images != nil {
		config, err = applyMerge(config, "images", *profile.Merge.Images)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.Dev != nil {
		config, err = applyMerge(config, "dev", *profile.Merge.Dev)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.Deployments != nil {
		config, err = applyMerge(config, "deployments", *profile.Merge.Deployments)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.OldDeployments != nil {
		config, err = applyMerge(config, "deployments", *profile.Merge.OldDeployments)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.Vars != nil {
		config, err = applyMerge(config, "vars", *profile.Merge.Vars)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.OldVars != nil {
		config, err = applyMerge(config, "vars", *profile.Merge.OldVars)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.Dependencies != nil {
		config, err = applyMerge(config, "dependencies", *profile.Merge.Dependencies)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.OldDependencies != nil {
		config, err = applyMerge(config, "dependencies", *profile.Merge.OldDependencies)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.PullSecrets != nil {
		config, err = applyMerge(config, "pullSecrets", *profile.Merge.PullSecrets)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.OldPullSecrets != nil {
		config, err = applyMerge(config, "pullSecrets", *profile.Merge.OldPullSecrets)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.Commands != nil {
		config, err = applyMerge(config, "commands", *profile.Merge.Commands)
		if err != nil {
			return nil, err
		}
	}
	if profile.Merge.OldCommands != nil {
		config, err = applyMerge(config, "commands", *profile.Merge.OldCommands)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func applyMerge(config map[string]interface{}, key string, value interface{}) (map[string]interface{}, error) {
	if value == nil {
		return config, nil
	}
	switch t := value.(type) {
	case []interface{}:
		if t == nil {
			return config, nil
		}
	case map[string]interface{}:
		if t == nil {
			return config, nil
		}
	}

	mergeObj := map[string]interface{}{
		key: value,
	}

	mergeBytes, err := json.Marshal(mergeObj)
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
func ApplyReplace(config map[string]interface{}, profile *latest.ProfileConfig) error {
	if profile == nil || profile.Replace == nil {
		return nil
	}

	if profile.Replace.Commands != nil {
		setKey(config, "commands", *profile.Replace.Commands)
	}
	if profile.Replace.OldCommands != nil {
		setKey(config, "commands", *profile.Replace.OldCommands)
	}
	if profile.Replace.Deployments != nil {
		setKey(config, "deployments", *profile.Replace.Deployments)
	}
	if profile.Replace.OldDeployments != nil {
		setKey(config, "deployments", *profile.Replace.OldDeployments)
	}
	if profile.Replace.Vars != nil {
		setKey(config, "vars", *profile.Replace.Vars)
	}
	if profile.Replace.OldVars != nil {
		setKey(config, "vars", *profile.Replace.OldVars)
	}
	if profile.Replace.Images != nil {
		setKey(config, "images", *profile.Replace.Images)
	}
	if profile.Replace.Dependencies != nil {
		setKey(config, "dependencies", *profile.Replace.Dependencies)
	}
	if profile.Replace.OldDependencies != nil {
		setKey(config, "dependencies", *profile.Replace.OldDependencies)
	}
	if profile.Replace.Dev != nil {
		setKey(config, "dev", *profile.Replace.Dev)
	}
	if profile.Replace.Hooks != nil {
		setKey(config, "hooks", *profile.Replace.Hooks)
	}
	if profile.Replace.PullSecrets != nil {
		setKey(config, "pullSecrets", *profile.Replace.PullSecrets)
	}
	if profile.Replace.OldPullSecrets != nil {
		setKey(config, "pullSecrets", *profile.Replace.OldPullSecrets)
	}
	return nil
}

func setKey(m map[string]interface{}, key string, value interface{}) {
	if value != nil {
		switch t := value.(type) {
		case []interface{}:
			if t == nil {
				return
			}
		case map[string]interface{}:
			if t == nil {
				return
			}
		}

		m[key] = value
	}
}

// ApplyPatches applies the patches to the config if defined
func ApplyPatches(data map[string]interface{}, profile *latest.ProfileConfig) (map[string]interface{}, error) {
	if profile == nil || len(profile.Patches) == 0 {
		return data, nil
	}

	return ApplyPatchesOnObject(data, profile.Patches)
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
			Path: patch.OpPath(patch.TransformPath(patchConfig.Path)),
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
