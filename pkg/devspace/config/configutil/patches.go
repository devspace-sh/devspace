package configutil

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/pkg/errors"

	yamlpatch "github.com/krishicks/yaml-patch"
	yaml "gopkg.in/yaml.v2"
)

// ApplyPatches applies the patches to the config if defined
func ApplyPatches(config *latest.Config) (*latest.Config, error) {
	out, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	// Check if there are patches defined
	if len(config.Profiles) == 1 && len(config.Profiles[0].Patches) > 0 {
		var iface map[interface{}]interface{}
		err := yaml.Unmarshal(out, &iface)
		if err != nil {
			return nil, err
		}

		var (
			patches = yamlpatch.Patch{}
		)

		for _, patch := range config.Profiles[0].Patches {
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
				target, _ := findPath(&newPatch.Path, iface)
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
	}

	newConfig := &latest.Config{}
	err = yaml.UnmarshalStrict(out, newConfig)
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
		log.Println(part)

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
