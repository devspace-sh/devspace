package build

import (
	"fmt"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/buildkit"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/docker"
	"github.com/loft-sh/devspace/pkg/devspace/build/registry"
	"github.com/loft-sh/devspace/pkg/devspace/build/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/loft-sh/devspace/pkg/util/stringutil"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/pkg/errors"
)

type imageNameAndTag struct {
	imageConfigName string
	imageName       string
	imageTag        string
	imageTags       []string
	imageConfig     latest.Image
}

// Options describe how images should be build
type Options struct {
	Tags                      []string `long:"tag" short:"t" description:"If enabled will override the default tags"`
	SkipBuild                 bool     `long:"skip" description:"If enabled will skip building"`
	SkipPush                  bool     `long:"skip-push" description:"Skip pushing"`
	SkipPushOnLocalKubernetes bool     `long:"skip-push-on-local-kubernetes" description:"Skip pushing"`
	ForceRebuild              bool     `long:"force-rebuild" description:"Skip pushing"`
	Sequential                bool     `long:"sequential" description:"Skip pushing"`

	MaxConcurrentBuilds int `long:"max-concurrent" description:"A pointer to an integer"`
}

// Controller is the main building interface
type Controller interface {
	Build(ctx devspacecontext.Context, images []string, options *Options) error
}

type controller struct{}

// NewController creates a new image build controller
func NewController() Controller {
	return &controller{}
}

// Build builds all images
func (c *controller) Build(ctx devspacecontext.Context, images []string, options *Options) error {
	var (
		builtImages = make(map[string]types.ImageNameTag)

		// Parallel build
		errChan   = make(chan error)
		cacheChan = make(chan imageNameAndTag)
		conf      = ctx.Config().Config()
	)

	// Check if we have at least 1 image to build
	if options.SkipBuild {
		ctx.Log().Debugf("Skip building because of --skip-build")
		return nil
	} else if len(conf.Images) == 0 {
		ctx.Log().Debugf("Skip building because no images are defined")
		return nil
	}

	// Build not in parallel when we only have one image to build
	if !options.Sequential {
		// check if all images are disabled besides one
		imagesToBuild := 0
		for k := range conf.Images {
			if len(images) > 0 && !stringutil.Contains(images, k) {
				continue
			}
			imagesToBuild++
		}
		if len(conf.Images) <= 1 || imagesToBuild <= 1 {
			options.Sequential = true
		}
	}

	// Determine if we need to use the local registry to build any images.
	var localRegistry *registry.LocalRegistry
	kubeClient := ctx.KubeClient()
	builders := map[string]builder.Interface{}
	tags := map[string][]string{}

	for imageConfigName, imageConf := range conf.Images {
		// create image pull secret if possible
		if ctx.KubeClient() != nil && (imageConf.CreatePullSecret == nil || *imageConf.CreatePullSecret) {
			registryURL, err := pullsecrets.GetRegistryFromImageName(imageConf.Image)
			if err != nil {
				return err
			}

			dockerClient, err := dockerclient.NewClient(ctx.Context(), ctx.Log())
			if err == nil {
				if imageConf.Kaniko != nil && imageConf.Kaniko.Namespace != "" && ctx.KubeClient().Namespace() != imageConf.Kaniko.Namespace {
					err = pullsecrets.NewClient().EnsurePullSecret(ctx, dockerClient, imageConf.Kaniko.Namespace, registryURL)
					if err != nil {
						ctx.Log().Errorf("error ensuring pull secret for registry %s: %v", registryURL, err)
					}
				}

				err = pullsecrets.NewClient().EnsurePullSecret(ctx, dockerClient, ctx.KubeClient().Namespace(), registryURL)
				if err != nil {
					ctx.Log().Errorf("error ensuring pull secret for registry %s: %v", registryURL, err)
				}
			}
		}

		// get image name and skip if we shouldn't build
		imageName := imageConf.Image
		if len(images) > 0 && !stringutil.Contains(images, imageConfigName) {
			continue
		}

		// Get image tags
		imageTags := []string{}
		if len(options.Tags) > 0 {
			imageTags = append(imageTags, options.Tags...)
		} else if len(imageConf.Tags) > 0 {
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
		builder, err := c.createBuilder(ctx, imageConfigName, imageConf, imageTags, options)
		if err != nil {
			return errors.Wrap(err, "create builder")
		}

		// Update cache for non local registry use by default
		imageCache, _ := ctx.Config().LocalCache().GetImageCache(imageConfigName)
		imageCache.ImageName = imageName
		imageCache.LocalRegistryImageName = ""

		if registry.UseLocalRegistry(kubeClient, conf, options.SkipPush) && !registry.HasPushPermission(imageConf) {
			if SupportsLocalRegistry(builder) {
				// Not able to deploy a local registry without a valid kube context
				if kubeClient == nil {
					return fmt.Errorf("unable to push image %s and a valid kube context is not available", imageConf.Image)
				}

				// Create and start a local registry if one isn't already running
				if localRegistry == nil {
					localRegistry = registry.NewLocalRegistry(
						registry.NewDefaultOptions().
							WithNamespace(kubeClient.Namespace()).
							WithLocalRegistryConfig(conf.LocalRegistry),
					)

					ctx := ctx.WithLogger(ctx.Log().WithPrefix("local-registry: "))
					err := localRegistry.Start(ctx)
					if err != nil {
						return errors.Wrap(err, "start registry")
					}
				}

				// Update cache for local registry use
				imageCache.LocalRegistryImageName, err = localRegistry.RewriteImage(imageName)
				if err != nil {
					return errors.Wrap(err, "rewrite image")
				}
				ctx.Config().LocalCache().SetImageCache(imageConfigName, imageCache)

				// Reset the builder for local registry usage
				// TODO: refactor so this isn't necessary!
				builder, err = c.createBuilder(ctx, imageConfigName, imageConf, imageTags, options)
				if err != nil {
					return errors.Wrap(err, "create builder")
				}
			} else {
				ctx.Log().Warnf("unable to push image %s and only docker and buildkit builds support using a local registry", imageConf.Image)
			}
		}

		// Save image cache
		ctx.Config().LocalCache().SetImageCache(imageConfigName, imageCache)

		// Save builder for later use
		builders[imageConfigName] = builder

		// Save image tags
		tags[imageConfigName] = imageTags
	}

	// Execute before images build hook
	pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{}, "before:build")
	if pluginErr != nil {
		return pluginErr
	}

	imagesToBuild := 0
	for key, imageConf := range conf.Images {
		ctx := ctx.WithLogger(ctx.Log().WithPrefix("build:" + key + " "))
		if len(images) > 0 && !stringutil.Contains(images, key) {
			continue
		}

		// This is necessary for parallel build otherwise we would override the image conf pointer during the loop
		cImageConf := *imageConf
		imageName := cImageConf.Image
		imageConfigName := key
		imageCache, _ := ctx.Config().LocalCache().GetImageCache(imageConfigName)
		resolvedImage := imageCache.ResolveImage()
		imageTags := tags[imageConfigName]
		builder := builders[imageConfigName]

		// Execute before images build hook
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"IMAGE_CONFIG_NAME": imageConfigName,
			"IMAGE_NAME":        resolvedImage,
			"IMAGE_CONFIG":      cImageConf,
			"IMAGE_TAGS":        imageTags,
		}, hook.EventsForSingle("before:build", imageConfigName).With("build.beforeBuild")...)
		if pluginErr != nil {
			return pluginErr
		}

		// Check if rebuild is needed
		needRebuild, err := builder.ShouldRebuild(ctx, options.ForceRebuild)
		if err != nil {
			pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
				"IMAGE_CONFIG_NAME": imageConfigName,
				"IMAGE_NAME":        resolvedImage,
				"IMAGE_CONFIG":      cImageConf,
				"IMAGE_TAGS":        imageTags,
				"ERROR":             err,
			}, hook.EventsForSingle("error:build", imageConfigName).With("build.errorBuild")...)
			if pluginErr != nil {
				return pluginErr
			}
			return errors.Errorf("error during shouldRebuild check: %v", err)
		}

		if !options.ForceRebuild && !needRebuild {
			// Execute before images build hook
			pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
				"IMAGE_CONFIG_NAME": imageConfigName,
				"IMAGE_NAME":        resolvedImage,
				"IMAGE_CONFIG":      cImageConf,
				"IMAGE_TAGS":        imageTags,
			}, hook.EventsForSingle("skip:build", imageConfigName)...)
			if pluginErr != nil {
				return pluginErr
			}
			ctx.Log().Infof("Skip building image '%s'", imageConfigName)
			continue
		}

		// Sequential or parallel build?
		if options.Sequential {
			// Build the image
			err = builder.Build(ctx)
			if err != nil {
				pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
					"IMAGE_CONFIG_NAME": imageConfigName,
					"IMAGE_NAME":        resolvedImage,
					"IMAGE_CONFIG":      cImageConf,
					"IMAGE_TAGS":        imageTags,
					"ERROR":             err,
				}, hook.EventsForSingle("error:build", imageConfigName).With("build.errorBuild")...)
				if pluginErr != nil {
					return pluginErr
				}
				return errors.Wrapf(err, "error building image %s:%s", resolvedImage, imageTags[0])
			}

			// Update cache
			imageCache, _ := ctx.Config().LocalCache().GetImageCache(imageConfigName)
			if imageCache.Tag == imageTags[0] {
				ctx.Log().Warnf("Newly built image '%s' has the same tag as in the last build (%s), this can lead to problems that the image during deployment is not updated", resolvedImage, imageTags[0])
			}

			imageCache.ImageName = imageName
			imageCache.Tag = imageTags[0]
			ctx.Config().LocalCache().SetImageCache(imageConfigName, imageCache)

			// Track built images
			builtImages[imageConfigName] = types.ImageNameTag{
				ImageConfigName: imageConfigName,
				ImageName:       imageName,
				ImageTag:        imageTags[0],
			}

			// Execute before images build hook
			pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
				"IMAGE_CONFIG_NAME": imageConfigName,
				"IMAGE_NAME":        resolvedImage,
				"IMAGE_CONFIG":      cImageConf,
				"IMAGE_TAGS":        imageTags,
			}, hook.EventsForSingle("after:build", imageConfigName).With("build.afterBuild")...)
			if pluginErr != nil {
				return pluginErr
			}
		} else {
			// wait until we are below the MaxConcurrency
			if options.MaxConcurrentBuilds > 0 && imagesToBuild >= options.MaxConcurrentBuilds {
				err = c.waitForBuild(ctx, errChan, cacheChan, builtImages)
				if err != nil {
					return err
				}

				imagesToBuild--
			}

			imagesToBuild++
			go func(ctx devspacecontext.Context) {
				// Build the image
				err := builder.Build(ctx)
				if err != nil {
					hook.LogExecuteHooks(ctx, map[string]interface{}{
						"IMAGE_CONFIG_NAME": imageConfigName,
						"IMAGE_NAME":        resolvedImage,
						"IMAGE_CONFIG":      cImageConf,
						"IMAGE_TAGS":        imageTags,
						"ERROR":             err,
					}, hook.EventsForSingle("error:build", imageConfigName).With("build.errorBuild")...)
					errChan <- errors.Errorf("error building image %s:%s: %v", resolvedImage, imageTags[0], err)
					return
				}

				// Send the response
				cacheChan <- imageNameAndTag{
					imageConfigName: imageConfigName,
					imageName:       imageName,
					imageTag:        imageTags[0],
					imageTags:       imageTags,
					imageConfig:     cImageConf,
				}
			}(ctx)
		}
	}

	// wait for the builds to finish
	if !options.Sequential {
		for imagesToBuild > 0 {
			err := c.waitForBuild(ctx, errChan, cacheChan, builtImages)
			if err != nil {
				return err
			}

			imagesToBuild--
		}
	}

	// Execute after images build hook
	pluginErr = hook.ExecuteHooks(ctx, map[string]interface{}{}, "after:build")
	if pluginErr != nil {
		return pluginErr
	}

	// merge built images
	alreadyBuiltImages, ok := ctx.Config().GetRuntimeVariable(constants.BuiltImagesKey)
	if ok {
		alreadyBuiltImagesMap, ok := alreadyBuiltImages.(map[string]types.ImageNameTag)
		if ok {
			for k, v := range alreadyBuiltImagesMap {
				_, ok := builtImages[k]
				if ok {
					continue
				}

				builtImages[k] = v
			}
		}
	}

	ctx.Config().SetRuntimeVariable(constants.BuiltImagesKey, builtImages)

	if len(builtImages) > 0 {
		err := ctx.Config().LocalCache().Save()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *controller) waitForBuild(ctx devspacecontext.Context, errChan <-chan error, cacheChan <-chan imageNameAndTag, builtImages map[string]types.ImageNameTag) error {
	select {
	case err := <-errChan:
		return err
	case done := <-cacheChan:
		imageCache, _ := ctx.Config().LocalCache().GetImageCache(done.imageConfigName)
		resolvedImage := imageCache.ResolveImage()

		ctx := ctx.WithLogger(ctx.Log().WithPrefix("build:" + done.imageConfigName + " "))
		ctx.Log().Donef("Done building image %s:%s (%s)", resolvedImage, done.imageTag, done.imageConfigName)

		// Update cache
		if imageCache.Tag == done.imageTag {
			ctx.Log().Warnf("Newly built image '%s' has the same tag as in the last build (%s), this can lead to problems that the image during deployment is not updated", done.imageName, done.imageTag)
		}

		imageCache.ImageName = done.imageName
		imageCache.Tag = done.imageTag
		ctx.Config().LocalCache().SetImageCache(done.imageConfigName, imageCache)

		// Track built images
		builtImages[done.imageConfigName] = types.ImageNameTag{
			ImageConfigName: done.imageConfigName,
			ImageName:       done.imageName,
			ImageTag:        done.imageTag,
		}

		// Execute plugin hook
		pluginErr := hook.ExecuteHooks(ctx, map[string]interface{}{
			"IMAGE_CONFIG_NAME": done.imageConfigName,
			"IMAGE_NAME":        resolvedImage,
			"IMAGE_CONFIG":      done.imageConfig,
			"IMAGE_TAGS":        done.imageTags,
		}, hook.EventsForSingle("after:build", done.imageConfigName).With("build.afterBuild")...)
		if pluginErr != nil {
			return pluginErr
		}
	}

	return nil
}

func SupportsLocalRegistry(builder builder.Interface) bool {
	switch builder.(type) {
	case *buildkit.Builder:
		return true
	case *docker.Builder:
		return true
	default:
		return false
	}
}
