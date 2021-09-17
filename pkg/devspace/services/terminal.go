package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	kubectlExec "k8s.io/client-go/util/exec"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	interruptpkg "github.com/loft-sh/devspace/pkg/util/interrupt"

	"github.com/mgutz/ansi"
)

type InterruptError struct{}

func (r *InterruptError) Error() string {
	return ""
}

// StartTerminal opens a new terminal
func (serviceClient *client) StartTerminal(
	options targetselector.Options,
	args []string,
	workDir string,
	interrupt chan error,
	wait,
	restart bool,
	stdout io.Writer,
	stderr io.Writer,
	stdin io.Reader,
) (int, error) {
	command := serviceClient.getCommand(args, workDir)
	targetSelector := targetselector.NewTargetSelector(serviceClient.client)
	if !wait {
		options.Wait = &wait
	} else {
		options.FilterPod = nil
		options.FilterContainer = nil
		options.WaitingStrategy = targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2)
	}
	options.Question = "Which pod do you want to open the terminal for?"

	container, err := targetSelector.SelectSingleContainer(context.TODO(), options, serviceClient.log)
	if err != nil {
		return 0, err
	}

	wrapper, upgradeRoundTripper, err := serviceClient.client.GetUpgraderWrapper()
	if err != nil {
		return 0, err
	}

	serviceClient.log.Infof("Opening shell to pod:container %s:%s", ansi.Color(container.Pod.Name, "white+b"), ansi.Color(container.Container.Name, "white+b"))

	done := make(chan error)
	go func() {
		interruptpkg.Global.Stop()
		defer interruptpkg.Global.Start()

		done <- serviceClient.client.ExecStreamWithTransport(&kubectl.ExecStreamWithTransportOptions{
			ExecStreamOptions: kubectl.ExecStreamOptions{
				Pod:       container.Pod,
				Container: container.Container.Name,
				Command:   command,
				TTY:       true,
				Stdin:     stdin,
				Stdout:    stdout,
				Stderr:    stderr,
			},
			Transport:   wrapper,
			Upgrader:    upgradeRoundTripper,
			SubResource: kubectl.SubResourceExec,
		})
	}()

	// wait until either client has finished or we got interrupted
	select {
	case err = <-interrupt:
		_ = upgradeRoundTripper.Close()
		<-done
		return 0, err
	case err = <-done:
		if err != nil {
			if _, ok := err.(*InterruptError); ok {
				return 0, err
			} else if exitError, ok := err.(kubectlExec.CodeExitError); ok {
				// Expected exit codes are (https://shapeshed.com/unix-exit-codes/):
				// 1 - Catchall for general errors
				// 2 - Misuse of shell builtins (according to Bash documentation)
				// 126 - Command invoked cannot execute
				// 127 - “command not found”
				// 128 - Invalid argument to exit
				// 130 - Script terminated by Control-C
				if restart && exitError.Code != 0 && exitError.Code != 1 && exitError.Code != 2 && exitError.Code != 126 && exitError.Code != 127 && exitError.Code != 128 && exitError.Code != 130 {
					serviceClient.log.WriteString("\n")
					serviceClient.log.Infof("Restarting terminal because: %s", err)
					return serviceClient.StartTerminal(options, args, workDir, interrupt, wait, restart, stdout, stderr, stdin)
				}

				return exitError.Code, nil
			} else if restart {
				serviceClient.log.WriteString("\n")
				serviceClient.log.Infof("Restarting terminal because: %s", err)
				return serviceClient.StartTerminal(options, args, workDir, interrupt, wait, restart, stdout, stderr, stdin)
			}

			return 0, err
		}
	}

	return 0, nil
}

func (serviceClient *client) getCommand(args []string, workDir string) []string {
	if serviceClient.config != nil && serviceClient.config.Config() != nil && serviceClient.config.Config().Dev.Terminal != nil {
		if len(args) == 0 {
			args = append(args, serviceClient.config.Config().Dev.Terminal.Command...)
		}
		if workDir == "" {
			workDir = serviceClient.config.Config().Dev.Terminal.WorkDir
		}
	}

	workDir = strings.TrimSpace(workDir)
	if len(args) > 0 {
		if workDir != "" {
			return []string{
				"sh",
				"-c",
				fmt.Sprintf("cd %s; %s", workDir, strings.Join(args, " ")),
			}
		}

		return args
	}

	execString := "command -v bash >/dev/null 2>&1 && exec bash || exec sh"
	if workDir != "" {
		execString = fmt.Sprintf("cd %s; %s", workDir, execString)
	}
	return []string{
		"sh",
		"-c",
		execString,
	}
}
