package loader

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/pkg/errors"

	yamlpatch "github.com/krishicks/yaml-patch"
	yaml "gopkg.in/yaml.v2"
)

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
				return nil, errors.Errorf("Cannot find key %s in object", part)
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
				foundContainer = iMap[part]
				continue
			}

			return nil, errors.Errorf("Unable to access invalid index: %d", i)
		}

		return nil, errors.Errorf("Cannot access part %s because value is not an object or array", part)
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
