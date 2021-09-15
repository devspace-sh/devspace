package services

import (
	"context"

	"github.com/pkg/errors"
)

func (serviceClient *client) ReplacePods() error {
	ctx := context.Background()
	for _, rp := range serviceClient.config.Config().Dev.ReplacePods {
		err := serviceClient.podReplacer.ReplacePod(ctx, serviceClient.client, serviceClient.config, serviceClient.dependencies, rp, serviceClient.log)
		if err != nil {
			return errors.Wrap(err, "replace pod")
		}
	}

	return nil
}
