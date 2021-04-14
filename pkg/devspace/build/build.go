package build

import (
	"bufio"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"io"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type imageNameAndTag struct {
	imageConfigName string
	imageName       string
	imageTag        string
}

// Options describe how images should be build
type Options struct {
	SkipPush                  bool
	SkipPushOnLocalKubernetes bool
	ForceRebuild              bool
	Sequential                bool
	MaxConcurrentBuilds       int
}

// Controller is the main building interface
type Controller interface {
	Build(options *Options, log logpkg.Logger) (map[string]string, error)
}

type controller struct {
	config config.Config

	hookExecuter hook.Executer
	client       kubectl.Client
}

// NewController creates a new image build controller
func NewController(config config.Config, dependencies []types.Dependency, client kubectl.Client) Controller {
	return &controller{
		config: config,

		hookExecuter: hook.NewExecuter(config, dependencies),
		client:       client,
	}
}

// All builds all images
func (c *controller) Build(options *Options, log logpkg.Logger) (map[string]string, error) {
	var (
		builtImages = make(map[string]string)

		// Parallel build
		errChan   = make(chan error)
		cacheChan = make(chan imageNameAndTag)
		config    = c.config.Config()
	)

	// Check if we have at least 1 image to build
	if len(config.Images) == 0 {
		return builtImages, nil
	}

	// Build not in parallel when we only have one image to build
	if options.Sequential == false {
		// check if all images are disabled besides one
		imagesToBuild := 0
		for _, image := range config.Images {
			if image.Build == nil || image.Build.Disabled == nil || *image.Build.Disabled == false {
				imagesToBuild++
			}
		}
		if len(config.Images) <= 1 || imagesToBuild <= 1 {
			options.Sequential = true
		}
	}

	// Execute before images build hook
	err := c.hookExecuter.Execute(hook.Before, hook.StageImages, hook.All, hook.Context{Client: c.client}, log)
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

		// Get image tags
		imageTags := []string{}
		if len(imageConf.Tags) > 0 {
			imageTags = append(imageTags, imageConf.Tags...)
		} else {
			imageTags = append(imageTags, randutil.GenerateRandomString(7))
		}

		// replace the # in the tags
		for i := range imageTags {
			for strings.Contains(imageTags[i], "#") {
				imageTags[i] = strings.Replace(imageTags[i], "#", randutil.GenerateRandomString(1), 1)
			}
		}

		// Create new builder
		builder, err := c.createBuilder(imageConfigName, &cImageConf, imageTags, options, log)
		if err != nil {
			return nil, errors.Wrap(err, "create builder")
		}

		// Check if rebuild is needed
		needRebuild, err := builder.ShouldRebuild(c.config.Generated().GetActive(), options.ForceRebuild)
		if err != nil {
			return nil, errors.Errorf("error during shouldRebuild check: %v", err)
		}

		if options.ForceRebuild == false && needRebuild == false {
			log.Infof("Skip building image '%s'", imageConfigName)
			continue
		}

		// Execute before images build hook
		err = c.hookExecuter.Execute(hook.Before, hook.StageImages, imageConfigName, hook.Context{Client: c.client}, log)
		if err != nil {
			return nil, err
		}

		// Sequential or parallel build?
		if options.Sequential {
			// Build the image
			err = builder.Build(log)
			if err != nil {
				c.hookExecuter.OnError(hook.StageImages, []string{hook.All, imageConfigName}, hook.Context{Client: c.client, Error: err}, log)
				return nil, errors.Wrapf(err, "error building image %s:%s", imageName, imageTags[0])
			}

			// Update cache
			imageCache := c.config.Generated().GetActive().GetImageCache(imageConfigName)
			if imageCache.Tag == imageTags[0] {
				log.Warnf("Newly built image '%s' has the same tag as in the last build (%s), this can lead to problems that the image during deployment is not updated", imageName, imageTags[0])
			}

			imageCache.ImageName = imageName
			imageCache.Tag = imageTags[0]

			// Track built images
			builtImages[imageName] = imageTags[0]

			// Execute before images build hook
			err = c.hookExecuter.Execute(hook.After, hook.StageImages, imageConfigName, hook.Context{Client: c.client}, log)
			if err != nil {
				return nil, err
			}
		} else {
			// wait until we are below the MaxConcurrency
			if options.MaxConcurrentBuilds > 0 && imagesToBuild >= options.MaxConcurrentBuilds {
				err = c.waitForBuild(errChan, cacheChan, builtImages, log)
				if err != nil {
					return nil, err
				}

				imagesToBuild--
			}

			imagesToBuild++
			go func() {
				// Create a string log
				reader, writer := io.Pipe()
				streamLog := logpkg.NewStreamLogger(writer, logrus.InfoLevel)
				logsLog := logpkg.NewPrefixLogger("["+imageConfigName+"] ", logpkg.Colors[(len(logpkg.Colors)-1)-(imagesToBuild%len(logpkg.Colors))], log)

				// read from the reader
				go func() {
					scanner := bufio.NewScanner(reader)
					for scanner.Scan() {
						logsLog.Info(scanner.Text())
					}
				}()

				// Build the image
				err := builder.Build(streamLog)
				_ = writer.Close()
				if err != nil {
					c.hookExecuter.OnError(hook.StageImages, []string{imageConfigName}, hook.Context{Client: c.client, Error: err}, log)
					errChan <- errors.Errorf("error building image %s:%s: %v", imageName, imageTags[0], err)
					return
				}

				// Execute before images build hook
				err = c.hookExecuter.Execute(hook.After, hook.StageImages, imageConfigName, hook.Context{Client: c.client}, log)
				if err != nil {
					errChan <- errors.Errorf("error executing image hook %s:%s: %v", imageName, imageTags[0], err)
					return
				}

				// Send the reponse
				cacheChan <- imageNameAndTag{
					imageConfigName: imageConfigName,
					imageName:       imageName,
					imageTag:        imageTags[0],
				}
			}()
		}
	}

	// wait for the builds to finish
	if options.Sequential == false {
		for imagesToBuild > 0 {
			err = c.waitForBuild(errChan, cacheChan, builtImages, log)
			if err != nil {
				return nil, err
			}

			imagesToBuild--
		}
	}

	// Execute after images build hook
	err = c.hookExecuter.Execute(hook.After, hook.StageImages, hook.All, hook.Context{Client: c.client}, log)
	if err != nil {
		return nil, err
	}

	return builtImages, nil
}

func (c *controller) waitForBuild(errChan <-chan error, cacheChan <-chan imageNameAndTag, builtImages map[string]string, log logpkg.Logger) error {
	select {
	case err := <-errChan:
		c.hookExecuter.OnError(hook.StageImages, []string{hook.All}, hook.Context{Client: c.client, Error: err}, log)
		return err
	case done := <-cacheChan:
		log.Donef("Done building image %s:%s (%s)", done.imageName, done.imageTag, done.imageConfigName)

		// Update cache
		imageCache := c.config.Generated().GetActive().GetImageCache(done.imageConfigName)
		if imageCache.Tag == done.imageTag {
			log.Warnf("Newly built image '%s' has the same tag as in the last build (%s), this can lead to problems that the image during deployment is not updated", done.imageName, done.imageTag)
		}

		imageCache.ImageName = done.imageName
		imageCache.Tag = done.imageTag

		// Track built images
		builtImages[done.imageName] = done.imageTag
	}

	return nil
}
