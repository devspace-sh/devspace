package util

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/devspace-cloud/devspace/pkg/devspace/pullsecrets"
)

func replaceImageNames(cache *generated.CacheConfig, imagesConf map[string]*latest.ImageConfig, builtImages map[string]string, keys map[string]bool, action func(walk.MatchFn, walk.ReplaceFn)) bool {
	if imagesConf == nil {
		imagesConf = map[string]*latest.ImageConfig{}
	}
	if keys == nil {
		keys = map[string]bool{}
	}

	// strip out images from cache that are not in the imagesconf anymore
	for name := range cache.Images {
		if _, ok := imagesConf[name]; !ok {
			delete(cache.Images, name)
		}
	}

	shouldRedeploy := false

	match := func(path, key, value string) bool {
		if len(keys) > 0 && keys[key] == false {
			return false
		}

		// Strip tag from image
		image, err := pullsecrets.GetStrippedDockerImageName(value)
		if err != nil {
			return false
		}

		// Search for image name
		for _, imageCache := range cache.Images {
			if imageCache.ImageName == image && imageCache.Tag != "" {
				if builtImages != nil {
					if _, ok := builtImages[image]; ok {
						shouldRedeploy = true
					}
				}

				return true
			}
		}

		return false
	}

	replace := func(path, value string) (interface{}, error) {
		image, err := pullsecrets.GetStrippedDockerImageName(value)
		if err != nil {
			return false, nil
		}

		// Search for image name
		for _, imageCache := range cache.Images {
			if imageCache.ImageName == image {
				return image + ":" + imageCache.Tag, nil
			}
		}

		return value, nil
	}

	// We ignore the error here because the replace function can never throw an error
	action(match, replace)

	return shouldRedeploy
}

func ReplaceImageNamesStringMap(manifest map[string]interface{}, cache *generated.CacheConfig, imagesConf map[string]*latest.ImageConfig, builtImages map[string]string, keys map[string]bool) bool {
	return replaceImageNames(cache, imagesConf, builtImages, keys, func(match walk.MatchFn, replace walk.ReplaceFn) {
		_ = walk.WalkStringMap(manifest, match, replace)
	})
}

// ReplaceImageNames replaces images within a certain manifest with the correct tags from the cache
func ReplaceImageNames(manifest map[interface{}]interface{}, cache *generated.CacheConfig, imagesConf map[string]*latest.ImageConfig, builtImages map[string]string, keys map[string]bool) bool {
	return replaceImageNames(cache, imagesConf, builtImages, keys, func(match walk.MatchFn, replace walk.ReplaceFn) {
		_ = walk.Walk(manifest, match, replace)
	})
}
