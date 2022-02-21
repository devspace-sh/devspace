package logs

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/mgutz/ansi"
	"io"
)

// StartLogsWithWriter prints the logs and then attaches to the container with the given stdout and stderr
func StartLogsWithWriter(ctx *devspacecontext.Context, selector targetselector.TargetSelector, follow bool, tail int64, writer io.Writer) error {
	container, err := selector.SelectSingleContainer(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return err
	}

	ctx.Log.Infof("Printing logs of pod:container %s:%s", ansi.Color(container.Pod.Name, "white+b"), ansi.Color(container.Container.Name, "white+b"))
	reader, err := ctx.KubeClient.Logs(ctx.Context, container.Pod.Namespace, container.Pod.Name, container.Container.Name, false, &tail, follow)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, reader)
	return err
}

// StartLogs print the logs and then attaches to the container
func StartLogs(ctx *devspacecontext.Context, devContainer *latest.DevContainer, selector targetselector.TargetSelector) error {
	container, err := selector.WithContainer(devContainer.Container).SelectSingleContainer(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return err
	}

	reader, err := ctx.KubeClient.Logs(ctx.Context, container.Pod.Namespace, container.Pod.Name, container.Container.Name, false, nil, true)
	if err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		s := scanner.NewScanner(reader)
		for s.Scan() {
			if devContainer.Container != "" {
				ctx.Log.Info(devContainer.Container + ": " + s.Text())
			} else {
				ctx.Log.Info(s.Text())
			}
		}

		errChan <- s.Err()
	}()

	select {
	case <-ctx.Context.Done():
		reader.Close()
		return nil
	case err := <-errChan:
		if err != nil {
			ctx.Log.WriteString("\n")
			ctx.Log.Infof("Restarting logs because: %s", err)
			return StartLogs(ctx, devContainer, selector)
		}

		return nil
	}
}
