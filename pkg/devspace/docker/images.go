package docker

import (
	"context"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
)

// DeleteImageByName deletes an image by name
func DeleteImageByName(client dockerclient.CommonAPIClient, imageName string, log log.Logger) ([]types.ImageDeleteResponseItem, error) {
	return DeleteImageByFilter(client, filters.NewArgs(filters.Arg("reference", strings.TrimSpace(imageName))), log)
}

// DeleteImageByFilter deletes an image by filter
func DeleteImageByFilter(client dockerclient.CommonAPIClient, filter filters.Args, log log.Logger) ([]types.ImageDeleteResponseItem, error) {
	summary, err := client.ImageList(context.Background(), types.ImageListOptions{
		Filters: filter,
	})
	if err != nil {
		return nil, err
	}

	responseItems := make([]types.ImageDeleteResponseItem, 0, 128)
	for _, image := range summary {
		deleteResponse, err := client.ImageRemove(context.Background(), image.ID, types.ImageRemoveOptions{
			PruneChildren: true,
			Force:         true,
		})
		if err != nil {
			log.Warnf("%v", err)
		}

		if deleteResponse != nil {
			responseItems = append(responseItems, deleteResponse...)
		}
	}

	return responseItems, nil
}
