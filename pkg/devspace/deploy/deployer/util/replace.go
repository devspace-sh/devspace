package util

import (
	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"regexp"
)

var (
	imageNameRegEx      = regexp.MustCompile(`^imageName\("?'?([^)"']+)"?'?\)$`)
	imageNameImageRegEx = regexp.MustCompile(`^imageNameImage\("?'?([^)"']+)"?'?\)$`)
	imageNameTagRegEx   = regexp.MustCompile(`^imageNameTag\("?'?([^)"']+)"?'?\)$`)
	imageRegEx          = regexp.MustCompile(`^image\("?'?([^)"']+)"?'?\)$`)
	tagRegEx            = regexp.MustCompile(`^tag\("?'?([^)"']+)"?'?\)$`)
)

func get(in string, regEx *regexp.Regexp) string {
	matches := regEx.FindStringSubmatch(in)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func match(key, value string, keys map[string]bool, config config2.Config) bool {
	if len(keys) > 0 && keys[key] == false {
		return false
	}

	// If we only want to set the tag or image
	var (
		onlyImage          = get(value, imageRegEx)
		onlyTag            = get(value, tagRegEx)
		imageName          = get(value, imageNameRegEx)
		imageNameOnlyImage = get(value, imageNameImageRegEx)
		imageNameOnlyTag   = get(value, imageNameTagRegEx)
	)
	switch {
	case onlyImage != "":
		value = onlyImage
	case onlyTag != "":
		value = onlyTag
	case imageName != "":
		return true
	case imageNameOnlyImage != "":
		return true
	case imageNameOnlyTag != "":
		return true
	}

	// Strip tag from image
	image, err := imageselector.GetStrippedDockerImageName(value)
	if err != nil {
		return false
	}

	// Search for image name
	for _, imageCache := range config.Generated().GetActive().Images {
		if imageCache.ImageName == image && imageCache.Tag != "" {
			return true
		}
	}

	return false
}

func replace(value string, config config2.Config, dependencies []types.Dependency, builtImages map[string]string) (bool, interface{}, error) {
	var (
		err                error
		selector           *imageselector.ImageSelector
		onlyImage          = get(value, imageRegEx)
		onlyTag            = get(value, tagRegEx)
		imageName          = get(value, imageNameRegEx)
		imageNameOnlyImage = get(value, imageNameImageRegEx)
		imageNameOnlyTag   = get(value, imageNameTagRegEx)
	)
	switch {
	case onlyImage != "":
		value = onlyImage
	case onlyTag != "":
		value = onlyTag
	case imageName != "":
		selector, err = imageselector.ResolveSingle(imageName, config, dependencies)
	case imageNameOnlyImage != "":
		selector, err = imageselector.ResolveSingle(imageNameOnlyImage, config, dependencies)
	case imageNameOnlyTag != "":
		selector, err = imageselector.ResolveSingle(imageNameOnlyTag, config, dependencies)
	}
	if err != nil {
		return false, nil, err
	} else if selector != nil {
		value = selector.Image
		if selector.Dependency != nil {
			config = selector.Dependency.Config()
			builtImages = selector.Dependency.BuiltImages()
		}
	}

	// ensure we don't run into any nil pointers
	config = config2.Ensure(config)

	// strip out images from cache that are not in the images conf anymore
	imageCache := config.Generated().GetActive().Images
	if imageCache == nil {
		imageCache = map[string]*generated.ImageCache{}
	}
	for name := range config.Config().Images {
		if _, ok := imageCache[name]; !ok {
			delete(imageCache, name)
		}
	}

	// strip docker image name
	image, err := imageselector.GetStrippedDockerImageName(value)
	if err != nil {
		return false, nil, nil
	}

	// check if in built images
	shouldRedeploy := false
	if builtImages != nil {
		if _, ok := builtImages[image]; ok {
			shouldRedeploy = true
		}
	}

	// only return the image
	if onlyImage != "" || imageNameOnlyImage != "" {
		return shouldRedeploy, image, nil
	}

	// Search for image name
	for _, imageCache := range imageCache {
		if imageCache.ImageName == image {
			if onlyTag != "" || imageNameOnlyTag != "" {
				return shouldRedeploy, imageCache.Tag, nil
			}

			return shouldRedeploy, image + ":" + imageCache.Tag, nil
		}
	}

	return shouldRedeploy, value, nil
}

func replaceImageNames(config config2.Config, dependencies []types.Dependency, builtImages map[string]string, keys map[string]bool, action func(walk.MatchFn, walk.ReplaceFn) error) (bool, error) {
	config = config2.Ensure(config)
	if keys == nil {
		keys = map[string]bool{}
	}

	shouldRedeploy := false
	err := action(func(key, value string) bool {
		return match(key, value, keys, config)
	}, func(value string) (interface{}, error) {
		redeploy, retValue, err := replace(value, config, dependencies, builtImages)
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
