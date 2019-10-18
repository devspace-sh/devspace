package cleanup

import (
	"context"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type imagesCmd struct {
	*flags.GlobalFlags
}

func newImagesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &imagesCmd{GlobalFlags: globalFlags}

	imagesCmd := &cobra.Command{
		Use:   "images",
		Short: "Deletes all locally created images from docker",
		Long: ` 
#######################################################
############# devspace cleanup images #################
#######################################################
Deletes all locally created docker images from docker
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: cmd.RunCleanupImages,
	}

	return imagesCmd
}

// RunCleanupImages executes the cleanup images command logic
func (cmd *imagesCmd) RunCleanupImages(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot(log.GetInstance())
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Get active context
	kubeContext, err := kubeconfig.GetCurrentContext()
	if err != nil {
		return err
	}
	if cmd.KubeContext != "" {
		kubeContext = cmd.KubeContext
	}

	// Create docker client
	client, err := docker.NewClientWithMinikube(kubeContext, true, log.GetInstance())
	if err != nil {
		return err
	}

	// Load config
	config, err := configutil.GetConfig(cmd.ToConfigOptions())
	if err != nil {
		return err
	}
	if config.Images == nil || len(config.Images) == 0 {
		log.Done("No images found in config to delete")
		return nil
	}

	_, err = client.Ping(context.Background())
	if err != nil {
		return errors.Errorf("Docker seems to be not running: %v", err)
	}

	defer log.StopWait()

	// Delete all images
	for _, imageConfig := range config.Images {
		log.StartWait("Deleting local image " + imageConfig.Image)

		response, err := docker.DeleteImageByName(client, imageConfig.Image, log.GetInstance())
		if err != nil {
			return err
		}

		for _, t := range response {
			if t.Deleted != "" {
				log.Donef("Deleted %s", t.Deleted)
			} else if t.Untagged != "" {
				log.Donef("Untagged %s", t.Untagged)
			}
		}
	}

	log.StartWait("Deleting local dangling images")

	// Cleanup dangling images aswell
	for {
		response, err := docker.DeleteImageByFilter(client, filters.NewArgs(filters.Arg("dangling", "true")), log.GetInstance())
		if err != nil {
			return err
		}

		for _, t := range response {
			if t.Deleted != "" {
				log.Donef("Deleted %s", t.Deleted)
			} else if t.Untagged != "" {
				log.Donef("Untagged %s", t.Untagged)
			}
		}

		if len(response) == 0 {
			break
		}
	}

	log.StopWait()
	log.Donef("Successfully cleaned up images")
	return nil
}
