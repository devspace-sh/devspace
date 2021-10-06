package services

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

func (serviceClient *client) ReplacePods(prefixFn PrefixFn) error {
	ctx := context.Background()
	runner := NewRunner(5)
	for idx, rp := range serviceClient.config.Config().Dev.ReplacePods {
		err := runner.Run(serviceClient.newReplacePodsFn(ctx, idx, rp, prefixFn))
		if err != nil {
			return err
		}
	}

	return runner.Wait()
}

func (serviceClient *client) newReplacePodsFn(ctx context.Context, idx int, rp *latest.ReplacePod, prefixFn PrefixFn) func() error {
	return func() error {
		prefix := prefixFn(idx, rp.Name, "replacePod")
		log := logpkg.NewUnionLogger(logpkg.NewDefaultPrefixLogger(prefix, serviceClient.log), logpkg.NewPrefixLogger(prefix, "", logpkg.GetFileLogger("replace-pods")))

		err := serviceClient.podReplacer.ReplacePod(ctx, serviceClient.client, serviceClient.config, serviceClient.dependencies, rp, log)
		if err != nil {
			return errors.Wrap(err, "replace pod")
		}

		return nil
	}
}
