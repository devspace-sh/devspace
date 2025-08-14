package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/pkg/errors"
	"time"
)

type WaitPodOptions struct {
	ImageSelector string `long:"image-selector" description:"The image selector to use to select the container"`
	LabelSelector string `long:"label-selector" description:"The label selector to use to select the container"`
	Container     string `long:"container" description:"The container to use"`

	Namespace   string `long:"namespace" short:"n" description:"The namespace to use"`
	DisableWait bool   `long:"disable-wait" description:"If true, will not wait for the container to become ready"`
	Timeout     int64  `long:"timeout" description:"The timeout to wait. Defaults to 5 minutes"`
}

func WaitPod(ctx devspacecontext.Context, args []string) error {
	if ctx.KubeClient() == nil {
		return errors.Errorf(ErrMsg)
	}
	options := &WaitPodOptions{
		Namespace: ctx.KubeClient().Namespace(),
	}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: wait_pod [--image-selector|--label-selector]")
	}
	if options.ImageSelector == "" && options.LabelSelector == "" {
		return fmt.Errorf("usage: wait_pod [--image-selector|--label-selector]")
	}

	logger := ctx.Log().ErrorStreamOnly()
	selectorOptions := targetselector.NewOptionsFromFlags(options.Container, options.LabelSelector, []string{options.ImageSelector}, options.Namespace, "")
	if options.Timeout != 0 {
		selectorOptions = selectorOptions.WithTimeout(options.Timeout)
	}
	selectorOptions.WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Millisecond * 100))
	_, err = targetselector.NewTargetSelector(selectorOptions).SelectSingleContainer(ctx.Context(), ctx.KubeClient(), logger)
	if err != nil {
		return err
	}

	return nil
}
