package runtime

import (
	"fmt"
	buildtypes "github.com/loft-sh/devspace/pkg/devspace/build/types"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"strings"
)

var Locations = []string{
	"/images/*/build/custom/command",
	"/images/*/build/custom/commands/*/command",
	"/images/*/build/custom/args/*",
	"/images/*/build/custom/appendArgs/*",
	"/deployments/*/helm/values/**",
	"/hooks/*/command",
	"/hooks/*/args/*",
	"/hooks/*/container/imageSelector",
	"/dev/*/imageSelector",
	"/dev/*/replaceImage",
	"/dev/*/containers/*/replaceImage",
	"/pipelines/**",
}

// NewRuntimeVariable creates a new variable that is loaded during runtime
func NewRuntimeVariable(name string, config config.Config, dependencies []types.Dependency) *runtimeVariable {
	return &runtimeVariable{
		name:         name,
		config:       config,
		dependencies: dependencies,
	}
}

type runtimeVariable struct {
	name         string
	config       config.Config
	dependencies []types.Dependency
}

func (e *runtimeVariable) Load() (bool, interface{}, error) {
	if !strings.HasPrefix(e.name, "runtime.") {
		return false, nil, fmt.Errorf("%s is no runtime variable", e.name)
	}

	runtimeVar := strings.TrimPrefix(e.name, "runtime.")
	c := e.config
	if strings.HasPrefix(runtimeVar, "dependencies.") {
		runtimeVar = strings.TrimPrefix(runtimeVar, "dependencies.")
		dependencyName := strings.Split(runtimeVar, ".")[0]
		if !strings.HasPrefix(runtimeVar, dependencyName+".") {
			return false, nil, fmt.Errorf("unexpected runtime variable %s, need format runtime.dependencies.NAME", e.name)
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

	runtimeVariables := c.ListRuntimeVariables()
	if runtimeVariables == nil {
		return false, nil, fmt.Errorf("couldn't find runtime variable %s", e.name)
	}

	// generic retrieve runtime variable
	out, ok := runtimeVariables[runtimeVar]
	if ok {
		return false, out, nil
	}

	// get image info from generated config
	if strings.HasPrefix(runtimeVar, "images.") {
		runtimeVar = strings.TrimPrefix(runtimeVar, "images.")
		if c.Config() == nil || c.LocalCache() == nil {
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
		cache := c.LocalCache()
		for configImageKey, configImage := range c.Config().Images {
			if configImageKey != imageName {
				continue
			}

			// check if in built images
			shouldRedeploy := false
			builtImagesInterface, ok := c.GetRuntimeVariable(constants.BuiltImagesKey)
			if ok {
				builtImages := builtImagesInterface.(map[string]buildtypes.ImageNameTag)
				for _, v := range builtImages {
					if v.ImageName == configImage.Image {
						shouldRedeploy = true
						break
					}
				}
			}

			// if we only need the image we are done here
			if onlyImage {
				return shouldRedeploy, configImage.Image, nil
			}

			// try to find the tag for the image
			tag := ""
			imageCache, _ := cache.GetImageCache(configImageKey)
			if imageCache.Tag != "" {
				tag = imageCache.Tag
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

	return false, nil, fmt.Errorf("couldn't find runtime variable %s", e.name)
}
