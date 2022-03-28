package docker

import (
	"context"
	"strings"

	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

// DeleteImageByName deletes an image by name
func (c *client) DeleteImageByName(ctx context.Context, imageName string, log log.Logger) ([]types.ImageDeleteResponseItem, error) {
	return c.DeleteImageByFilter(ctx, filters.NewArgs(filters.Arg("reference", strings.TrimSpace(imageName))), log)
}

// DeleteImageByFilter deletes an image by filter
func (c *client) DeleteImageByFilter(ctx context.Context, filter filters.Args, log log.Logger) ([]types.ImageDeleteResponseItem, error) {
	summary, err := c.ImageList(ctx, types.ImageListOptions{
		Filters: filter,
	})
	if err != nil {
		return nil, err
	}

	responseItems := make([]types.ImageDeleteResponseItem, 0, 128)
	for _, image := range summary {
		deleteResponse, err := c.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{
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
