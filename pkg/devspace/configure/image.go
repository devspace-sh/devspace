package configure

import (
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/util/log"
)

//AddImage adds an image to the devspace
func AddImage(nameInConfig, name, tag, contextPath, dockerfilePath, buildEngine string) error {
	config := configutil.GetBaseConfig()

	imageConfig := &v1.ImageConfig{
		Image: &name,
		Build: &v1.BuildConfig{},
	}

	if tag != "" {
		imageConfig.Tag = &tag
	}
	if contextPath != "" {
		imageConfig.Build.ContextPath = &contextPath
	}
	if dockerfilePath != "" {
		imageConfig.Build.DockerfilePath = &dockerfilePath
	}

	if buildEngine == "docker" {
		imageConfig.Build.Docker = &v1.DockerConfig{}
	} else if buildEngine == "kaniko" {
		imageConfig.Build.Kaniko = &v1.KanikoConfig{}
	} else if buildEngine != "" {
		log.Errorf("BuildEngine %v unknown. Please select one of docker|kaniko", buildEngine)
	}

	if config.Images == nil {
		images := make(map[string]*v1.ImageConfig)
		config.Images = &images
	}

	(*config.Images)[nameInConfig] = imageConfig

	err := configutil.SaveBaseConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %s", err.Error())
	}

	return nil
}

//RemoveImage removes an image from the devspace
func RemoveImage(removeAll bool, names []string) error {
	config := configutil.GetBaseConfig()

	if len(names) == 0 && removeAll == false {
		return fmt.Errorf("You have to specify at least one image")
	}

	newImageList := make(map[string]*v1.ImageConfig)

	if !removeAll && config.Images != nil {

	ImagesLoop:
		for nameInConfig, imageConfig := range *config.Images {
			for _, deletionName := range names {
				if deletionName == nameInConfig {
					continue ImagesLoop
				}
			}

			newImageList[nameInConfig] = imageConfig
		}
	}

	config.Images = &newImageList

	err := configutil.SaveBaseConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %v", err)
	}

	return nil
}
