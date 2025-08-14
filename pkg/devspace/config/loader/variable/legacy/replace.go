package legacy

import (
	"fmt"
	"regexp"
	"strings"

	buildtypes "github.com/loft-sh/devspace/pkg/devspace/build/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"github.com/loft-sh/devspace/pkg/util/dockerfile"

	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
)

var (
	imageRegEx = regexp.MustCompile(`(?m)image\("?'?([^)"']+)"?'?\)`)
	tagRegEx   = regexp.MustCompile(`(?m)tag\("?'?([^)"']+)"?'?\)`)
)

type replaceFn func(match string) (bool, bool, string, error)

func replaceWithRegEx(in string, replaceFn replaceFn, regEx *regexp.Regexp) (bool, string, error) {
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

		found, shouldRedeploy, resolvedImage, err := replaceFn(match[1])
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
	if len(keys) > 0 && !keys[key] {
		return false
	}

	return true
}

func resolveImage(value string, config config2.Config, dependencies []types.Dependency, tryImageKey, onlyImage, onlyTag bool) (bool, bool, string, error) {
	builtImages := map[string]buildtypes.ImageNameTag{}
	builtImagesInterface, ok := config.GetRuntimeVariable(constants.BuiltImagesKey)
	if ok {
		builtImages, _ = builtImagesInterface.(map[string]buildtypes.ImageNameTag)
	}

	resolvedImage := value
	if tryImageKey {
		selector, err := imageselector.Resolve(value, config, dependencies)
		if err == nil && selector != nil {
			resolvedImage = selector.Image
			if selector.Dependency != nil {
				config = selector.Dependency.Config()

				builtImagesInterface, ok := selector.Dependency.Config().GetRuntimeVariable(constants.BuiltImagesKey)
				if ok {
					builtImages, _ = builtImagesInterface.(map[string]buildtypes.ImageNameTag)
				} else {
					builtImages = map[string]buildtypes.ImageNameTag{}
				}
			}
		}
	}

	// ensure we don't run into any nil pointers
	config = config2.Ensure(config)

	// strip out images from cache that are not in the images conf anymore
	imageCacheMap := config.LocalCache().ListImageCache()

	// Find matching image in image cache
	var imageCache *localcache.ImageCache
	if tryImageKey {
		if match, ok := imageCacheMap[value]; ok {
			imageCache = &match
		}
	} else {
		for _, match := range imageCacheMap {
			if match.ImageName == value {
				imageCache = &match
				break
			}
		}
	}

	// strip original image name
	originalImage := ""
	if imageCache != nil {
		strippedImage, _, err := dockerfile.GetStrippedDockerImageName(imageCache.ImageName)
		if err != nil {
			return false, false, "", nil
		}
		originalImage = strippedImage
	}

	// strip docker image name
	image, originalTag, err := dockerfile.GetStrippedDockerImageName(resolvedImage)
	if err != nil {
		return false, false, "", nil
	}

	// check if in built images
	shouldRedeploy := false
	for _, v := range builtImages {
		if v.ImageName == image || v.ImageName == originalImage {
			shouldRedeploy = true
			break
		}
	}

	// Found a match in the image cache
	if imageCache != nil {
		if onlyImage {
			return true, shouldRedeploy, imageCache.ResolveImage(), nil
		}

		if onlyTag {
			if imageCache.Tag == "" {
				return true, shouldRedeploy, "latest", nil
			}

			return true, shouldRedeploy, imageCache.Tag, nil
		}

		return true, shouldRedeploy, imageCache.ResolveImage() + ":" + imageCache.Tag, nil
	}

	// config images
	configImages := config.Config().Images
	if configImages == nil {
		configImages = map[string]*latest.Image{}
	}

	// search for image name
	for configImageKey, configImage := range configImages {
		if configImage.Image != image && configImage.Image != originalImage {
			continue
		}

		effectiveImage := image
		imageCache, ok := imageCacheMap[configImageKey]
		if ok && imageCache.LocalRegistryImageName != "" {
			effectiveImage = imageCache.LocalRegistryImageName
		}

		// if we only need the image we are done here
		if onlyImage {
			return true, shouldRedeploy, effectiveImage, nil
		}

		// try to find the tag for the image
		tag := originalTag
		if imageCache.Tag != "" {
			tag = imageCache.Tag
		}

		// does the config have a tag defined?
		if tag == "" && len(configImage.Tags) > 0 {
			tag = strings.ReplaceAll(configImage.Tags[0], "#", "x")
		}

		// only return the tag
		if onlyTag {
			if tag == "" {
				return true, shouldRedeploy, "latest", nil
			}

			return true, shouldRedeploy, tag, nil
		}

		// return either with or without tag
		if tag == "" {
			return true, shouldRedeploy, effectiveImage, nil
		}

		return true, shouldRedeploy, effectiveImage + ":" + tag, nil
	}

	// not found, return the initial value
	return false, shouldRedeploy, value, nil
}

func ResolveImageHelpers(value string, config config2.Config, dependencies []types.Dependency) (string, error) {
	_, image, err := ReplaceHelpers(value, config, dependencies)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", image), nil
}

func ResolveImage(imageSelector string, config config2.Config, dependencies []types.Dependency) (string, error) {
	_, image, err := Replace(imageSelector, config, dependencies)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", image), nil
}

func ResolveImageAsImageSelector(imageSelector string, config config2.Config, dependencies []types.Dependency) (*imageselector.ImageSelector, error) {
	image, err := ResolveImage(imageSelector, config, dependencies)
	if err != nil {
		return nil, err
	}

	return &imageselector.ImageSelector{
		Image: image,
	}, nil
}

func Replace(value string, config config2.Config, dependencies []types.Dependency) (bool, interface{}, error) {
	// check if it's just a single image name
	found, shouldRedeploy, resolvedImage, err := resolveImage(value, config, dependencies, false, false, false)
	if err != nil {
		return false, nil, err
	} else if found {
		return shouldRedeploy, resolvedImage, nil
	}

	helperShouldRedeploy, helperResolvedImage, err := ReplaceHelpers(value, config, dependencies)
	return shouldRedeploy || helperShouldRedeploy, helperResolvedImage, err
}

func ReplaceHelpers(value string, config config2.Config, dependencies []types.Dependency) (bool, interface{}, error) {
	// replace the image() helpers
	shouldRedeploy, value, err := replaceWithRegEx(value, func(match string) (bool, bool, string, error) {
		return resolveImage(match, config, dependencies, true, true, false)
	}, imageRegEx)
	if err != nil {
		return false, nil, err
	}

	// replace the tag() helpers
	imageShouldRedeploy := shouldRedeploy
	shouldRedeploy, value, err = replaceWithRegEx(value, func(match string) (bool, bool, string, error) {
		return resolveImage(match, config, dependencies, true, false, true)
	}, tagRegEx)
	if err != nil {
		return false, nil, err
	}

	return imageShouldRedeploy || shouldRedeploy, value, nil
}

func replaceImageNames(config config2.Config, dependencies []types.Dependency, keys map[string]bool, action func(walk.MatchFn, walk.ReplaceFn) error) (bool, error) {
	config = config2.Ensure(config)
	if keys == nil {
		keys = map[string]bool{}
	}

	shouldRedeploy := false
	err := action(func(key, value string) bool {
		return Match(key, value, keys)
	}, func(_, value string) (interface{}, error) {
		redeploy, retValue, err := Replace(value, config, dependencies)
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

func ReplaceImageNamesStringMap(manifest map[string]interface{}, config config2.Config, dependencies []types.Dependency, keys map[string]bool) (bool, error) {
	return replaceImageNames(config, dependencies, keys, func(match walk.MatchFn, replace walk.ReplaceFn) error {
		return walk.WalkStringMap(manifest, match, replace)
	})
}

// ReplaceImageNames replaces images within a certain manifest with the correct tags from the cache
func ReplaceImageNames(manifest map[string]interface{}, config config2.Config, dependencies []types.Dependency, keys map[string]bool) (bool, error) {
	return replaceImageNames(config, dependencies, keys, func(match walk.MatchFn, replace walk.ReplaceFn) error {
		return walk.Walk(manifest, match, replace)
	})
}
