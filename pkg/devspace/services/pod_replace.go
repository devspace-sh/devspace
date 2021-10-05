package services

import (
	"context"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

func (serviceClient *client) ReplacePods(prefixFn PrefixFn) error {
	ctx := context.Background()
	runner := NewRunner(5)
	for idx, rp := range serviceClient.config.Config().Dev.ReplacePods {
		err := runner.Run(func() error {
			prefix := prefixFn(idx, rp.Name, "replacePod")
			log := logpkg.NewUnionLogger(logpkg.NewDefaultPrefixLogger(prefix, serviceClient.log), logpkg.NewPrefixLogger(prefix, "", logpkg.GetFileLogger("replace-pods")))

			err := serviceClient.podReplacer.ReplacePod(ctx, serviceClient.client, serviceClient.config, serviceClient.dependencies, rp, log)
			if err != nil {
				return errors.Wrap(err, "replace pod")
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return runner.Wait()
}
