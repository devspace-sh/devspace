package imageselector

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/util/dockerfile"
)

type ImageSelector struct {
	// Image is the resolved docker image inclusive tag
	Image string
	// Dependency is the dependency this image selector was loaded from
	Dependency types.Dependency
}

func Resolve(configImageName string, config config.Config, dependencies []types.Dependency) (*ImageSelector, error) {
	if configImageName != "" && config != nil && config.LocalCache() != nil && config.Config() != nil {
		var (
			c         = config.Config()
			generated = config.LocalCache()
		)

		// check if cached
		imageCache, _ := generated.GetImageCache(configImageName)
		if imageCache.ImageName != "" && imageCache.Tag != "" && c.Images != nil && c.Images[configImageName] != nil {
			return &ImageSelector{
				Image: imageCache.ResolveImage() + ":" + imageCache.Tag,
			}, nil
		}

		// check if defined in images
		if c.Images != nil && c.Images[configImageName] != nil {
			if len(c.Images[configImageName].Tags) > 0 {
				return &ImageSelector{
					Image: c.Images[configImageName].Image + ":" + strings.ReplaceAll(c.Images[configImageName].Tags[0], "#", "x"),
				}, nil
			}

			return &ImageSelector{
				Image: c.Images[configImageName].Image,
			}, nil
		}

		// check if image from dependency
		if strings.Contains(configImageName, ".") {
			dependency := configImageName[:strings.Index(configImageName, ".")]
			dependencyImageName := configImageName[len(dependency)+1:]

			for _, dep := range dependencies {
				if dep.Name() == dependency {
					imageSelector, err := Resolve(dependencyImageName, dep.Config(), dep.Children())
					if err != nil {
						return nil, err
					} else if imageSelector == nil {
						return imageSelector, nil
					}

					// if no dependency is set, we set it here
					if imageSelector.Dependency == nil {
						imageSelector.Dependency = dep
					}

					// make sure the selector has the correct original name
					return imageSelector, nil
				}
			}
		}

		return nil, fmt.Errorf("couldn't find imageName %s", configImageName)
	}

	return nil, nil
}

func CompareImageNames(selector string, image2 string) bool {
	image1 := selector

	// we replace possible # with a's here to avoid an parsing error
	// since the tag is stripped anyways it doesn't really matter if we lose
	// information where the # were
	tagStrippedImage1, _, err := dockerfile.GetStrippedDockerImageName(strings.ReplaceAll(image1, "#", "a"))
	if err != nil {
		tagStrippedImage1 = image1
	}
	tagStrippedImage2, _, err := dockerfile.GetStrippedDockerImageName(image2)
	if err != nil {
		tagStrippedImage2 = image2
	}

	if tagStrippedImage1 != image1 {
		// In the case that the tag is latest and we find an image that has no tag
		if tagStrippedImage1+":latest" == image1 && tagStrippedImage2 == image2 {
			return true
		}

		// if the tag consists of a # we build a regex
		if strings.Contains(image1, "#") {
			regex := "^" + strings.ReplaceAll(image1, "#", "[a-zA-Z]") + "$"
			exp, err := regexp.Compile(regex)
			if err == nil {
				return exp.MatchString(image2)
			}
		}

		return image1 == image2
	}

	return tagStrippedImage1 == tagStrippedImage2
}
