package attach

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/terminal"
	interruptpkg "github.com/loft-sh/devspace/pkg/util/interrupt"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/sirupsen/logrus"
	"io"
	kubectlExec "k8s.io/client-go/util/exec"
	"os"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/mgutz/ansi"
)

// StartAttachFromCMD opens a new terminal
func StartAttachFromCMD(ctx devspacecontext.Context, selector targetselector.TargetSelector) error {
	container, err := selector.SelectSingleContainer(ctx.Context(), ctx.KubeClient(), ctx.Log())
	if err != nil {
		return err
	}

	if !container.Container.TTY || !container.Container.Stdin {
		ctx.Log().Warnf("To be able to interact with the container its options tty (currently `%t`) and stdin (currently `%t`) must both be `true`", container.Container.TTY, container.Container.Stdin)
	}

	ctx.Log().Infof("Attaching to pod:container %s:%s", ansi.Color(container.Pod.Name, "white+b"), ansi.Color(container.Container.Name, "white+b"))
	ctx.Log().Info("If you don't see a command prompt, try pressing enter.")

	done := make(chan error)
	go func() {
		done <- ctx.KubeClient().ExecStream(ctx.Context(), &kubectl.ExecStreamOptions{
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
	case <-ctx.Context().Done():
		<-done
		return nil
	case err := <-done:
		return err
	}
}

// StartAttach opens a new terminal
func StartAttach(
	ctx devspacecontext.Context,
	devContainer *latest.DevContainer,
	selector targetselector.TargetSelector,
	stdout io.Writer,
	stderr io.Writer,
	stdin io.Reader,
	parent *tomb.Tomb,
) (err error) {
	// restart on error
	defer func() {
		if err != nil {
			if ctx.IsDone() {
				return
			}

			ctx.Log().Infof("Restarting because: %s", err)
			select {
			case <-ctx.Context().Done():
				return
			case <-time.After(time.Second * 3):
			}
			err = StartAttach(ctx, devContainer, selector, stdout, stderr, stdin, parent)
			return
		}

		ctx.Log().Debugf("Stopped attach")
	}()

	before := log.GetBaseInstance().GetLevel()
	log.GetBaseInstance().SetLevel(logrus.PanicLevel)
	defer log.GetBaseInstance().SetLevel(before)

	container, err := selector.WithContainer(devContainer.Container).SelectSingleContainer(ctx.Context(), ctx.KubeClient(), ctx.Log())
	if err != nil {
		return err
	}

	ctx.Log().Infof("Attaching to pod:container %s:%s", ansi.Color(container.Pod.Name, "white+b"), ansi.Color(container.Container.Name, "white+b"))
	errChan := make(chan error)
	parent.Go(func() error {
		interruptpkg.Global.Stop()
		defer interruptpkg.Global.Start()

		errChan <- ctx.KubeClient().ExecStream(ctx.Context(), &kubectl.ExecStreamOptions{
			Pod:         container.Pod,
			Container:   container.Container.Name,
			TTY:         container.Container.TTY,
			Stdin:       stdin,
			Stdout:      stdout,
			Stderr:      stderr,
			SubResource: kubectl.SubResourceAttach,
		})
		return nil
	})

	select {
	case <-ctx.Context().Done():
		<-errChan
		return nil
	case err = <-errChan:
		if ctx.IsDone() {
			return nil
		}

		if err != nil {
			// check if context is done
			if exitError, ok := err.(kubectlExec.CodeExitError); ok {
				// Expected exit codes are (https://shapeshed.com/unix-exit-codes/):
				// 1 - Catchall for general errors
				// 2 - Misuse of shell builtins (according to Bash documentation)
				// 126 - Command invoked cannot execute
				// 127 - “command not found”
				// 128 - Invalid argument to exit
				// 130 - Script terminated by Control-C
				if terminal.IsUnexpectedExitCode(exitError.Code) {
					return err
				}

				return nil
			}

			return fmt.Errorf("lost connection to pod %s: %v", container.Pod.Name, err)
		}
	}

	return nil
}
