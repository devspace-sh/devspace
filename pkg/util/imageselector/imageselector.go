package imageselector

import (
	"fmt"
	"github.com/docker/distribution/reference"
	dockerregistry "github.com/docker/docker/registry"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"regexp"
	"strings"
)

type ImageSelector struct {
	// ConfigImageName is the image config name (from images.*)
	ConfigImageName string
	// Image is the resolved docker image
	Image string
}

func Resolve(configImageName string, config config.Config, dependencies []types.Dependency) ([]ImageSelector, error) {
	if configImageName != "" && config != nil && config.Generated() != nil && config.Config() != nil {
		var (
			c         = config.Config()
			generated = config.Generated().GetActive()
		)

		// check if cached
		imageConfigCache := generated.GetImageCache(configImageName)
		if imageConfigCache.ImageName != "" {
			return []ImageSelector{
				{
					ConfigImageName: configImageName,
					Image:           imageConfigCache.ImageName + ":" + imageConfigCache.Tag,
				},
			}, nil
		}

		// check if defined in images
		if c.Images[configImageName] != nil {
			if len(c.Images[configImageName].Tags) > 0 {
				return []ImageSelector{
					{
						ConfigImageName: configImageName,
						Image:           c.Images[configImageName].Image + ":" + c.Images[configImageName].Tags[0],
					},
				}, nil
			}

			return []ImageSelector{
				{
					ConfigImageName: configImageName,
					Image:           c.Images[configImageName].Image,
				},
			}, nil
		}

		// check if image from dependency
		splitted := strings.Split(configImageName, ".")
		if len(splitted) == 2 {
			for _, dep := range dependencies {
				if dep.DependencyConfig().Name == splitted[0] {
					imageSelector, err := Resolve(splitted[1], dep.Config(), nil)
					if err != nil {
						return nil, err
					} else if len(imageSelector) != 1 {
						return imageSelector, nil
					}

					// make sure the selector has the correct original name
					imageSelector[0].ConfigImageName = configImageName
					return imageSelector, nil
				}
			}
		}

		return nil, fmt.Errorf("couldn't find imageName %s", configImageName)
	}

	return []ImageSelector{}, nil
}

func CompareImageNames(selector ImageSelector, image2 string) bool {
	image1 := selector.Image

	// we replace possible # with a's here to avoid an parsing error
	// since the tag is stripped anyways it doesn't really matter if we lose
	// information where the # were
	tagStrippedImage1, err := GetStrippedDockerImageName(strings.Replace(image1, "#", "a", -1))
	if err != nil {
		tagStrippedImage1 = image1
	}
	tagStrippedImage2, err := GetStrippedDockerImageName(image2)
	if err != nil {
		tagStrippedImage2 = image2
	}

	if tagStrippedImage1 != image1 {
		// In the case that the tag is latest and we find an image that has no tag
		if tagStrippedImage1+":latest" == image1 && tagStrippedImage2 == image2 {
			return true
		}

		// if the tag consists of a # we build a regex
		if strings.Index(image1, "#") != -1 {
			regex := "^" + strings.Replace(image1, "#", "[a-zA-Z]", -1) + "$"
			exp, err := regexp.Compile(regex)
			if err == nil {
				return exp.MatchString(image2)
			}
		}

		return image1 == image2
	} else {
		return tagStrippedImage1 == tagStrippedImage2
	}
}

// GetStrippedDockerImageName returns a tag stripped image name and checks if it's a valid image name
func GetStrippedDockerImageName(imageName string) (string, error) {
	imageName = strings.TrimSpace(imageName)

	// Check if we can parse the name
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return "", err
	}

	repoInfo, err := dockerregistry.ParseRepositoryInfo(ref)
	if err != nil {
		return "", err
	}

	if repoInfo.Index.Official {
		// strip docker.io and library from image
		return strings.TrimPrefix(strings.TrimPrefix(reference.TrimNamed(ref).Name(), repoInfo.Index.Name+"/library/"), repoInfo.Index.Name+"/"), nil
	}

	return reference.TrimNamed(ref).Name(), nil
}
