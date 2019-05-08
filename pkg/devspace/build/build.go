package build

import (
	"bytes"
	"fmt"

	"k8s.io/client-go/kubernetes"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type imageNameAndTag struct {
	imageConfigName string
	imageName       string
	imageTag        string
}

// All builds all images
func All(config *latest.Config, client kubernetes.Interface, isDev, forceRebuild, sequential bool, log logpkg.Logger) (map[string]string, error) {
	var (
		builtImages = make(map[string]string)

		// Parallel build
		errChan   = make(chan error)
		cacheChan = make(chan imageNameAndTag)
	)

	// Build not in parallel when we only have one image to build
	if sequential == false && len(*config.Images) <= 1 {
		sequential = true
	}

	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		return nil, errors.Wrap(err, "load generated config")
	}

	// Update config
	cache := generatedConfig.GetActive()

	imagesToBuild := 0
	for key, imageConf := range *config.Images {
		if imageConf.Build != nil && imageConf.Build.Disabled != nil && *imageConf.Build.Disabled == true {
			log.Infof("Skipping building image %s", key)
			continue
		}

		// This is necessary for parallel build otherwise we would override the image conf pointer during the loop
		cImageConf := *imageConf
		imageName := *cImageConf.Image
		imageConfigName := key

		// Get image tag
		imageTag, err := randutil.GenerateRandomString(7)
		if err != nil {
			return nil, fmt.Errorf("Image building failed: %v", err)
		}
		if imageConf.Tag != nil {
			imageTag = *imageConf.Tag
		}

		// Create new builder
		builder, err := CreateBuilder(config, client, imageConfigName, &cImageConf, imageTag, isDev, log)
		if err != nil {
			return nil, errors.Wrap(err, "create builder")
		}

		// Check if rebuild is needed
		needRebuild, err := builder.ShouldRebuild(cache)
		if err != nil {
			return nil, fmt.Errorf("Error during shouldRebuild check: %v", err)
		}
		if forceRebuild == false && needRebuild == false {
			log.Infof("Skip building image '%s'", imageConfigName)
			continue
		}

		// Sequential or parallel build?
		if sequential {
			// Build the image
			err = builder.Build(log)
			if err != nil {
				return nil, err
			}

			// Update cache
			imageCache := cache.GetImageCache(imageConfigName)
			imageCache.ImageName = imageName
			imageCache.Tag = imageTag

			// Track built images
			builtImages[imageName] = imageTag
		} else {
			imagesToBuild++
			go func() {
				// Create a string log
				buff := &bytes.Buffer{}
				streamLog := logpkg.NewStreamLogger(buff, logrus.InfoLevel)

				// Build the image
				err := builder.Build(streamLog)
				if err != nil {
					errChan <- fmt.Errorf("Error building image %s:%s: %s %v", imageName, imageTag, buff.String(), err)
					return
				}

				// Send the reponse
				cacheChan <- imageNameAndTag{
					imageConfigName: imageConfigName,
					imageName:       imageName,
					imageTag:        imageTag,
				}
			}()
		}
	}

	if sequential == false && imagesToBuild > 0 {
		defer log.StopWait()

		for imagesToBuild > 0 {
			log.StartWait(fmt.Sprintf("Building %d images...", imagesToBuild))

			select {
			case err := <-errChan:
				return nil, err
			case done := <-cacheChan:
				imagesToBuild--
				log.Donef("Done building image %s:%s (%s)", done.imageName, done.imageTag, done.imageConfigName)

				// Update cache
				imageCache := cache.GetImageCache(done.imageConfigName)
				imageCache.ImageName = done.imageName
				imageCache.Tag = done.imageTag

				// Track built images
				builtImages[done.imageName] = done.imageTag
			}
		}
	}

	return builtImages, nil
}
