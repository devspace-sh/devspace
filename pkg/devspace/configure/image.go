package configure

import (
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
)

//AddImage adds an image to the devspace
func AddImage(nameInConfig string, name string, tag string, contextPath string, dockerfilePath, buildEngine string) error {
	config := configutil.GetConfig()

	imageConfig := &v1.ImageConfig{
		Name: &name,
		Tag:  &tag,
		Build: &v1.BuildConfig{
			ContextPath:    &contextPath,
			DockerfilePath: &dockerfilePath,
		},
	}

	if buildEngine == "docker" {
		imageConfig.Build.Docker = &v1.DockerConfig{}
	} else if buildEngine == "kaniko" {
		imageConfig.Build.Kaniko = &v1.KanikoConfig{}
	} else if buildEngine != "" {
		log.Errorf("BuildEngine %v unknown. Please select one of docker|kaniko", buildEngine)
	}

	(*config.Images)[nameInConfig] = imageConfig

	err := configutil.SaveConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %s", err.Error())
	}

	return nil
}

//RemoveImage removes an image from the devspace
func RemoveImage(removeAll bool, names []string) error {
	config := configutil.GetConfig()

	if len(names) == 0 && removeAll == false {
		return fmt.Errorf("You have to specify at least one image")
	}

	newImageList := make(map[string]*v1.ImageConfig)

	if !removeAll {

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

	err := configutil.SaveConfig()
	if err != nil {
		return fmt.Errorf("Couldn't save config file: %v", err)
	}

	return nil
}
