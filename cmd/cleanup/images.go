package cleanup

import (
	"context"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/docker/docker/api/types/filters"
	"github.com/spf13/cobra"
)

type imagesCmd struct {
}

func newImagesCmd() *cobra.Command {
	cmd := &imagesCmd{}

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
		Run:  cmd.RunCleanupImages,
	}

	return imagesCmd
}

// RunCleanupImages executes the cleanup images command logic
func (cmd *imagesCmd) RunCleanupImages(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Load config
	config := configutil.GetConfig()
	if config.Images == nil || len(*config.Images) == 0 {
		log.Done("No images found in config to delete")
		return
	}

	// Get active context
	kubeContext, err := kubeconfig.GetCurrentContext()
	if err != nil {
		log.Fatal(err)
	}

	// Create docker client
	client, err := docker.NewClientWithMinikube(kubeContext, true, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.Ping(context.Background())
	if err != nil {
		log.Fatalf("Docker seems to be not running: %v", err)
	}

	defer log.StopWait()

	// Delete all images
	for _, imageConfig := range *config.Images {
		log.StartWait("Deleting local image " + *imageConfig.Image)

		response, err := docker.DeleteImageByName(client, *imageConfig.Image, log.GetInstance())
		if err != nil {
			log.Fatal(err)
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
			log.Fatal(err)
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
}
