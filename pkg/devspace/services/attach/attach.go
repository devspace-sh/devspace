package attach

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/mgutz/ansi"
)

// StartAttach opens a new terminal
func StartAttach(ctx *devspacecontext.Context, selector targetselector.TargetSelector) error {
	container, err := selector.SelectSingleContainer(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return err
	}

	if !container.Container.TTY || !container.Container.Stdin {
		ctx.Log.Warnf("To be able to interact with the container its options tty (currently `%t`) and stdin (currently `%t`) must both be `true`", container.Container.TTY, container.Container.Stdin)
	}

	ctx.Log.Infof("Attaching to pod:container %s:%s", ansi.Color(container.Pod.Name, "white+b"), ansi.Color(container.Container.Name, "white+b"))
	ctx.Log.Info("If you don't see a command prompt, try pressing enter.")

	done := make(chan error)
	go func() {
		done <- ctx.KubeClient.ExecStream(ctx.Context, &kubectl.ExecStreamOptions{
			Pod:         container.Pod,
			Container:   container.Container.Name,
			TTY:         container.Container.TTY,
			Stdin:       os.Stdin,
			Stdout:      os.Stdout,
			Stderr:      os.Stderr,
			SubResource: kubectl.SubResourceAttach,
		})
	}()

	// wait until either client has finished or we got interrupted
	select {
	case <-ctx.Context.Done():
		<-done
		return nil
	case err := <-done:
		return err
	}
}
