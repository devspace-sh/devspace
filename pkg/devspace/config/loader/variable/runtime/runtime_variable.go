package runtime

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"strings"
)

// NewRuntimeVariable creates a new variable that is loaded during runtime
func NewRuntimeVariable(name string, config config.Config, dependencies []types.Dependency, builtImages map[string]string) *runtimeVariable {
	return &runtimeVariable{
		name:         name,
		config:       config,
		dependencies: dependencies,
		builtImages:  builtImages,
	}
}

type runtimeVariable struct {
	name         string
	config       config.Config
	dependencies []types.Dependency
	builtImages  map[string]string
}

func (e *runtimeVariable) Load() (bool, interface{}, error) {
	if !strings.HasPrefix(e.name, "runtime.") {
		return false, nil, fmt.Errorf("%s is no runtime variable", e.name)
	}

	runtimeVar := strings.TrimPrefix(e.name, "runtime.")
	c := e.config
	if strings.HasPrefix(runtimeVar, "dependency.") {
		runtimeVar = strings.TrimPrefix(runtimeVar, "dependency.")
		dependencyName := strings.Split(runtimeVar, ".")[0]
		if !strings.HasPrefix(runtimeVar, dependencyName+".") {
			return false, nil, fmt.Errorf("unexpected runtime variable %s, need format runtime.dependency.NAME", e.name)
		}
		runtimeVar = strings.TrimPrefix(runtimeVar, dependencyName+".")

		found := false
		for _, dep := range e.dependencies {
			if dep.Name() == dependencyName {
				c = dep.Config()
				found = true
				break
			}
		}
		if !found {
			return false, nil, fmt.Errorf("couldn't find runtime variable %s, make sure the dependency %s was loaded", e.name, dependencyName)
		}
	}

	runtimeVariables := c.RuntimeVariables()
	if runtimeVariables == nil {
		return false, nil, fmt.Errorf("couldn't find runtime variable %s", e.name)
	}

	// get image info from generated config
	if strings.HasPrefix(runtimeVar, "images.") {
		runtimeVar = strings.TrimPrefix(runtimeVar, "images.")
		if c.Config() == nil || c.Generated() == nil {
			return false, nil, fmt.Errorf("couldn't find runtime variable %s, because config or cache is empty", e.name)
		}

		imageName := runtimeVar
		onlyImage := false
		onlyTag := false
		if strings.HasSuffix(runtimeVar, ".tag") {
			imageName = strings.TrimSuffix(runtimeVar, ".tag")
			onlyTag = true
		} else if strings.HasSuffix(runtimeVar, ".image") {
			imageName = strings.TrimSuffix(runtimeVar, ".image")
			onlyImage = true
		}

		// search for image name
		generated := c.Generated().GetActive()
		for configImageKey, configImage := range c.Config().Images {
			if configImageKey != imageName {
				continue
			}

			// check if in built images
			shouldRedeploy := false
			if e.builtImages != nil {
				if _, ok := e.builtImages[configImage.Image]; ok {
					shouldRedeploy = true
				}
			}

			// if we only need the image we are done here
			if onlyImage {
				return shouldRedeploy, configImage.Image, nil
			}

			// try to find the tag for the image
			tag := ""
			if generated.Images[configImageKey] != nil && generated.Images[configImageKey].Tag != "" {
				tag = generated.Images[configImageKey].Tag
			}

			// does the config have a tag defined?
			if tag == "" && len(configImage.Tags) > 0 {
				tag = strings.Replace(configImage.Tags[0], "#", "x", -1)
			}

			// only return the tag
			if onlyTag {
				if tag == "" {
					return shouldRedeploy, "latest", nil
				}

				return shouldRedeploy, tag, nil
			}

			// return either with or without tag
			if tag == "" {
				return shouldRedeploy, configImage.Image, nil
			}

			return shouldRedeploy, configImage.Image + ":" + tag, nil
		}

		return false, nil, fmt.Errorf("couldn't find imageName %s resolving variable %s", imageName, e.name)
	}

	// generic retrieve runtime variable
	out, ok := runtimeVariables[runtimeVar]
	if !ok {
		return false, nil, fmt.Errorf("couldn't find runtime variable %s", e.name)
	}

	return false, out, nil
}
