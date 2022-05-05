package commands

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/interp"
	"time"
)

type SelectPodOptions struct {
	ImageSelector string `long:"image-selector" description:"The image selector to use to select the container"`
	LabelSelector string `long:"label-selector" description:"The label selector to use to select the container"`
	Container     string `long:"container" description:"The container to use"`

	Namespace   string `long:"namespace" short:"n" description:"The namespace to use"`
	DisableWait bool   `long:"disable-wait" description:"If true, will not wait for the container to become ready"`
	Timeout     int64  `long:"timeout" description:"The timeout to wait. Defaults to 5 minutes"`
}

func SelectPod(ctx devspacecontext.Context, args []string) error {
	hc := interp.HandlerCtx(ctx.Context())
	if ctx.KubeClient() == nil {
		return errors.Errorf(ErrMsg)
	}
	options := &SelectPodOptions{
		Namespace: ctx.KubeClient().Namespace(),
	}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: select_pod [--image-selector|--label-selector]")
	}
	if options.ImageSelector == "" && options.LabelSelector == "" {
		return fmt.Errorf("usage: select_pod [--image-selector|--label-selector]")
	}

	logger := ctx.Log().ErrorStreamOnly()
	selectorOptions := targetselector.NewOptionsFromFlags(options.Container, options.LabelSelector, []string{options.ImageSelector}, options.Namespace, "")
	if options.Timeout != 0 {
		selectorOptions = selectorOptions.WithTimeout(options.Timeout)
	}
	selectorOptions.WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Millisecond * 100))
	selectedContainer, err := targetselector.NewTargetSelector(selectorOptions).SelectSingleContainer(ctx.Context(), ctx.KubeClient(), logger)
	if err != nil {
		return err
	}

	_, _ = hc.Stdout.Write([]byte(selectedContainer.Pod.Name))
	return nil
}
