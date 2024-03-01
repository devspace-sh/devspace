package runtime

import (
	"fmt"
	"strings"

	buildtypes "github.com/loft-sh/devspace/pkg/devspace/build/types"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/pkg/errors"
)

var Locations = []string{
	"/images/*/build/custom/command",
	"/images/*/build/custom/commands/*/command",
	"/images/*/build/custom/args/**",
	"/images/*/build/custom/appendArgs/**",
	"/deployments/*/helm/values/**",
	"/deployments/*/tanka/**",
	"/deployments/*/kubectl/inlineManifest/**",
	"/hooks/*/command",
	"/hooks/*/args/*",
	"/hooks/*/container/imageSelector",
	"/dev/*/imageSelector",
	"/dev/*/replaceImage",
	"/dev/*/devImage",
	"/dev/*/containers/*/replaceImage",
	"/dev/*/containers/*/devImage",
	"/dev/ports/*/imageSelector",
	"/dev/sync/*/imageSelector",
	"/dev/logs/*/selectors/*/imageSelector",
	"/dev/replacePods/*/imageSelector",
	"/dev/replacePods/*/replaceImage",
	"/dev/terminal/imageSelector",
	"/pipelines/*",
	"/pipelines/*/flags/**",
	"/pipelines/*/run",
	"/commands/*",
	"/commands/*/command",
	"/functions/**",
	"/imports/**",
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

		shouldRebuild, image, err := GetImage(c, imageName, onlyImage, onlyTag)
		if err != nil {
			return false, nil, errors.Wrapf(err, "resolving variable %s", e.name)
		}

		return shouldRebuild, image, nil
	}

	return false, nil, fmt.Errorf("couldn't find runtime variable %s", e.name)
}

func GetImage(c config.Config, imageName string, onlyImage, onlyTag bool) (bool, string, error) {
	// search for image name in cache
	imageCache, ok := c.LocalCache().GetImageCache(imageName)
	if ok && imageCache.ImageName != "" && imageCache.Tag != "" {
		shouldRedeploy, image := BuildImageString(c, imageName, imageCache.ImageName, imageCache.Tag, onlyImage, onlyTag)
		if image != "" {
			return shouldRedeploy, image, nil
		}
	}

	// search for image name in config
	if c.Config().Images != nil && c.Config().Images[imageName] != nil {
		configImage := c.Config().Images[imageName]

		tag := ""
		if len(configImage.Tags) > 0 {
			tag = configImage.Tags[0]
		}

		shouldRedeploy, image := BuildImageString(c, imageName, configImage.Image, tag, onlyImage, onlyTag)
		if image != "" {
			return shouldRedeploy, image, nil
		}
	}

	return false, "", fmt.Errorf("couldn't find imageName %s", imageName)
}

func BuildImageString(c config.Config, name string, fallbackImage string, fallbackTag string, onlyImage, onlyTag bool) (bool, string) {
	imageCache, _ := c.LocalCache().GetImageCache(name)

	// try to find the image
	image := imageCache.ResolveImage()
	if image == "" && fallbackImage != "" {
		image = fallbackImage
	}

	if image == "" {
		return false, ""
	}

	// check if in built images
	shouldRedeploy := false
	builtImagesInterface, ok := c.GetRuntimeVariable(constants.BuiltImagesKey)
	if ok {
		builtImages := builtImagesInterface.(map[string]buildtypes.ImageNameTag)
		_, found := builtImages[name]
		if found {
			shouldRedeploy = true
		}
	}

	// if we only need the image we are done here
	if onlyImage {
		return shouldRedeploy, image
	}

	// try to find the tag for the image
	tag := ""
	if imageCache.Tag != "" {
		tag = imageCache.Tag
	} else if fallbackTag != "" {
		tag = strings.ReplaceAll(fallbackTag, "#", "x")
	}

	// only return the tag
	if onlyTag {
		if tag == "" {
			return shouldRedeploy, "latest"
		}

		return shouldRedeploy, tag
	}

	// return either with or without tag
	if tag == "" {
		return shouldRedeploy, image
	}

	return shouldRedeploy, image + ":" + tag
}
