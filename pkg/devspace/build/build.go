package build

import (
	"bytes"
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/hook"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
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
func All(config *latest.Config, cache *generated.CacheConfig, client kubectl.Client, skipPush, isDev, forceRebuild, sequential, ignoreContextPathChanges bool, log logpkg.Logger) (map[string]string, error) {
	var (
		builtImages = make(map[string]string)

		// Parallel build
		errChan   = make(chan error)
		cacheChan = make(chan imageNameAndTag)
	)

	// Check if we have at least 1 image to build
	if len(config.Images) == 0 {
		return builtImages, nil
	}

	// Build not in parallel when we only have one image to build
	if sequential == false && len(config.Images) <= 1 {
		sequential = true
	}

	// Execute before images build hook
	executer := hook.NewExecuter(config, log)
	err := executer.Execute(hook.Before, hook.StageImages, hook.All)
	if err != nil {
		return nil, err
	}

	imagesToBuild := 0
	for key, imageConf := range config.Images {
		if imageConf.Build != nil && imageConf.Build.Disabled != nil && *imageConf.Build.Disabled == true {
			log.Infof("Skipping building image %s", key)
			continue
		}

		// This is necessary for parallel build otherwise we would override the image conf pointer during the loop
		cImageConf := *imageConf
		imageName := cImageConf.Image
		imageConfigName := key

		// Get image tag
		imageTag, err := randutil.GenerateRandomString(7)
		if err != nil {
			return nil, errors.Errorf("Image building failed: %v", err)
		}
		if imageConf.Tag != "" {
			imageTag = imageConf.Tag
		}

		// Create new builder
		builder, err := CreateBuilder(config, client, imageConfigName, &cImageConf, imageTag, skipPush, isDev, log)
		if err != nil {
			return nil, errors.Wrap(err, "create builder")
		}

		// Check if rebuild is needed
		needRebuild, err := builder.ShouldRebuild(cache, ignoreContextPathChanges)
		if err != nil {
			return nil, errors.Errorf("Error during shouldRebuild check: %v", err)
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
			if imageCache.Tag == imageTag {
				log.Warnf("Newly built image '%s' has the same tag as in the last build (%s), this can lead to problems that the image during deployment is not updated", imageName, imageTag)
			}

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
					errChan <- errors.Errorf("Error building image %s:%s: %s %v", imageName, imageTag, buff.String(), err)
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
				if imageCache.Tag == done.imageTag {
					log.Warnf("Newly built image '%s' has the same tag as in the last build (%s), this can lead to problems that the image during deployment is not updated", done.imageName, done.imageTag)
				}

				imageCache.ImageName = done.imageName
				imageCache.Tag = done.imageTag

				// Track built images
				builtImages[done.imageName] = done.imageTag
			}
		}
	}

	// Execute after images build hook
	err = executer.Execute(hook.After, hook.StageImages, hook.All)
	if err != nil {
		return nil, err
	}

	return builtImages, nil
}
