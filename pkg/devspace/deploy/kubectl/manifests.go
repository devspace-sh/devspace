package kubectl

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/covexo/devspace/pkg/util/log"

	yaml "gopkg.in/yaml.v2"
)

// Manifest is the type that holds the data of a single kubernetes manifest
type Manifest map[interface{}]interface{}

func joinManifests(manifests []Manifest) (string, error) {
	retString := ""

	for _, manifest := range manifests {
		out, err := yaml.Marshal(manifest)
		if err != nil {
			return "", nil
		}

		if retString != "" {
			retString += "\n---\n"
		}

		retString += string(out)
	}

	return retString, nil
}

func loadManifests(globPatterns []string, log log.Logger) ([]Manifest, error) {
	manifests := []Manifest{}

	for _, pattern := range globPatterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			if isValidFile(file) {
				loadedManifests, err := getManifestsFromFile(file)
				if err != nil {
					return nil, err
				}

				manifests = append(manifests, loadedManifests...)
			} else {
				log.Warnf("Manifest %s skipped because it does not have a valid ending (.yml or .yaml expected)", file)
			}
		}
	}

	return manifests, nil
}

func isValidFile(filepath string) bool {
	return strings.HasSuffix(filepath, ".yml") || strings.HasSuffix(filepath, ".yaml")
}

func getManifestsFromFile(filepath string) ([]Manifest, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	retManifests := []Manifest{}
	manifests := strings.Split(string(data), "\n---")

	for _, manifest := range manifests {
		manifest = strings.TrimLeft(manifest, "\r\n")
		manifest = strings.TrimRight(manifest, "\r\n")

		manifestData := Manifest{}

		err = yaml.Unmarshal([]byte(manifest), manifestData)
		if err != nil {
			return nil, err
		}

		retManifests = append(retManifests, manifestData)
	}

	return retManifests, nil
}
