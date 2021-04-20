package util

import (
	"fmt"
	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"regexp"
	"strings"
)

var (
	imageRegEx = regexp.MustCompile(`image\("?'?([^)"']+)"?'?\)`)
	tagRegEx   = regexp.MustCompile(`tag\("?'?([^)"']+)"?'?\)`)
)

func replaceWithRegEx(in string, config config2.Config, dependencies []types.Dependency, builtImages map[string]string, regEx *regexp.Regexp, onlyImage, onlyTag bool) (bool, string, error) {
	matches := regEx.FindAllStringSubmatch(in, -1)
	if len(matches) == 0 {
		return false, in, nil
	}

	out := in
	shouldRedeployTotal := false
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}

		found, shouldRedeploy, resolvedImage, err := resolveImage(match[1], config, dependencies, builtImages, true, onlyImage, onlyTag)
		if err != nil {
			return false, "", err
		} else if !found {
			continue
		}

		if shouldRedeploy {
			shouldRedeployTotal = true
		}

		out = strings.Replace(out, match[0], resolvedImage, 1)
	}

	return shouldRedeployTotal, out, nil
}

func Match(key, value string, keys map[string]bool) bool {
	if len(keys) > 0 && keys[key] == false {
		return false
	}

	return true
}

func resolveImage(value string, config config2.Config, dependencies []types.Dependency, builtImages map[string]string, tryImageKey, onlyImage, onlyTag bool) (bool, bool, string, error) {
	resolvedImage := value
	if tryImageKey {
		selector, err := imageselector.ResolveSingle(value, config, dependencies)
		if err == nil && selector != nil {
			resolvedImage = selector.Image
			if selector.Dependency != nil {
				config = selector.Dependency.Config()
				builtImages = selector.Dependency.BuiltImages()
			}
		}
	}

	// ensure we don't run into any nil pointers
	config = config2.Ensure(config)

	// strip out images from cache that are not in the images conf anymore
	imageCache := config.Generated().GetActive().Images
	if imageCache == nil {
		imageCache = map[string]*generated.ImageCache{}
	}

	// config images
	configImages := config.Config().Images
	if configImages == nil {
		configImages = map[string]*latest.ImageConfig{}
	}

	// strip docker image name
	image, err := imageselector.GetStrippedDockerImageName(resolvedImage)
	if err != nil {
		return false, false, "", nil
	}

	// check if in built images
	shouldRedeploy := false
	if builtImages != nil {
		if _, ok := builtImages[image]; ok {
			shouldRedeploy = true
		}
	}
	
	// search for image name
	for configImageKey, configImage := range configImages {
		if configImage.Image != image {
			continue
		}
		
		// if we only need the image we are done here
		if onlyImage {
			return true, shouldRedeploy, configImage.Image, nil
		}

		// try to find the tag for the image
		tag := ""
		if imageCache[configImageKey] != nil && imageCache[configImageKey].Tag != "" {
			tag = imageCache[configImageKey].Tag
		}

		// does the config have a tag defined?
		if tag == "" && len(configImage.Tags) > 0 {
			tag = configImage.Tags[0]
		}

		// only return the tag
		if onlyTag {
			if tag == "" {
				return false, false, "", fmt.Errorf("no tag found for image %s", image)
			}

			return true, shouldRedeploy, tag, nil
		}

		// return either with or without tag
		if tag == "" {
			return true, shouldRedeploy, image, nil
		}
		return true, shouldRedeploy, image + ":" + imageCache[configImageKey].Tag, nil
	}

	// not found, return the initial value
	return false, shouldRedeploy, value, nil
}

func Replace(value string, config config2.Config, dependencies []types.Dependency, builtImages map[string]string) (bool, interface{}, error) {
	// check if it's just a single image name
	found, shouldRedeploy, resolvedImage, err := resolveImage(value, config, dependencies, builtImages, false, false, false)
	if err != nil {
		return false, nil, err
	} else if found {
		return shouldRedeploy, resolvedImage, nil
	}

	// replace the image() helpers
	shouldRedeploy, value, err = replaceWithRegEx(value, config, dependencies, builtImages, imageRegEx, true, false)
	if err != nil {
		return false, nil, err
	}

	// replace the tag() helpers
	imageShouldRedeploy := shouldRedeploy
	shouldRedeploy, value, err = replaceWithRegEx(value, config, dependencies, builtImages, tagRegEx, false, true)
	if err != nil {
		return false, nil, err
	}

	return imageShouldRedeploy || shouldRedeploy, value, nil
}

func replaceImageNames(config config2.Config, dependencies []types.Dependency, builtImages map[string]string, keys map[string]bool, action func(walk.MatchFn, walk.ReplaceFn) error) (bool, error) {
	config = config2.Ensure(config)
	if keys == nil {
		keys = map[string]bool{}
	}

	shouldRedeploy := false
	err := action(func(key, value string) bool {
		return Match(key, value, keys)
	}, func(value string) (interface{}, error) {
		redeploy, retValue, err := Replace(value, config, dependencies, builtImages)
		if err != nil {
			return nil, err
		} else if redeploy {
			shouldRedeploy = redeploy
		}

		return retValue, nil
	})
	if err != nil {
		return false, err
	}

	return shouldRedeploy, nil
}

func ReplaceImageNamesStringMap(manifest map[string]interface{}, config config2.Config, dependencies []types.Dependency, builtImages map[string]string, keys map[string]bool) (bool, error) {
	return replaceImageNames(config, dependencies, builtImages, keys, func(match walk.MatchFn, replace walk.ReplaceFn) error {
		return walk.WalkStringMap(manifest, match, replace)
	})
}

// ReplaceImageNames replaces images within a certain manifest with the correct tags from the cache
func ReplaceImageNames(manifest map[interface{}]interface{}, config config2.Config, dependencies []types.Dependency, builtImages map[string]string, keys map[string]bool) (bool, error) {
	return replaceImageNames(config, dependencies, builtImages, keys, func(match walk.MatchFn, replace walk.ReplaceFn) error {
		return walk.Walk(manifest, match, replace)
	})
}
