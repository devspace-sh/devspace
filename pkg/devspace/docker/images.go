package docker

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/docker/docker/api/types/filters"
)

// DeleteImageByName deletes an image by name
func (c *client) DeleteImageByName(ctx context.Context, imageName string, log log.Logger) ([]image.DeleteResponse, error) {
	return c.DeleteImageByFilter(ctx, filters.NewArgs(filters.Arg("reference", strings.TrimSpace(imageName))), log)
}

// DeleteImageByFilter deletes an image by filter
func (c *client) DeleteImageByFilter(ctx context.Context, filter filters.Args, log log.Logger) ([]image.DeleteResponse, error) {
	summaries, err := c.ImageList(ctx, image.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return nil, err
	}

	responseItems := make([]image.DeleteResponse, 0, 128)
	for _, summary := range summaries {
		deleteResponse, err := c.ImageRemove(ctx, summary.ID, image.RemoveOptions{
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
