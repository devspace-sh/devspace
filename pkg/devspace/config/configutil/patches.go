package configutil

import (
	"regexp"
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
		patches := yamlpatch.Patch{}
		for _, patch := range config.Profiles[0].Patches {
			newPatch := yamlpatch.Operation{
				Op:   yamlpatch.Op(patch.Operation),
				Path: yamlpatch.OpPath(transformPath(patch.Path)),
				From: yamlpatch.OpPath(transformPath(patch.From)),
			}

			if patch.Value != nil {
				newPatch.Value = yamlpatch.NewNode(&patch.Value)
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
