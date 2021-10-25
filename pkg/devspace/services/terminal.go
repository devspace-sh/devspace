package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	kubectlExec "k8s.io/client-go/util/exec"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
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
	subcommand string,
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
		options.WaitingStrategy = targetselector.NewUntilNewestRunningWaitingStrategy(time.Second)
	}
	options.Question = "Which pod do you want to open the terminal for?"

	container, err := targetSelector.SelectSingleContainer(context.TODO(), options, serviceClient.log)
	if err != nil {
		return 0, err
	}

	podContainer := fmt.Sprintf("pod:container %s:%s", ansi.Color(container.Pod.Name, "white+b"), ansi.Color(container.Container.Name, "white+b"))

	// if terminal is invoked by devspace dev then install screen command for persistent session
	if subcommand == "dev" {
		// Install screen command
		serviceClient.log.Debugf("Installing screen command in pod:container %s", podContainer)
		screenInstalled, err := installScreen(container, serviceClient.client)
		if err != nil {
			serviceClient.log.Debugf("Failed to install screen command in %s, error: %v", podContainer, err)
		} else {
			serviceClient.log.Done("Successfully installed screen command %s", podContainer)
		}

		// if screen is installed then it starts the screen session
		if screenInstalled {
			serviceClient.log.Infof("Opening a screen persistent session into %s", podContainer)
			cmd := getScreenCommand(container, serviceClient.client, command)
			if cmd == nil {
				serviceClient.log.Errorf("Failed to get screen command")
				serviceClient.log.Infof("Opening shell to %s", podContainer)
			} else {
				command = cmd
			}
		}
	} else {
		serviceClient.log.Infof("Opening shell to %s", podContainer)
	}

	wrapper, upgradeRoundTripper, err := serviceClient.client.GetUpgraderWrapper()
	if err != nil {
		return 0, err
	}

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
				if restart && IsUnexpectedExitCode(exitError.Code) {
					serviceClient.log.WriteString("\n")
					serviceClient.log.Infof("Restarting terminal because: %s", err)
					return serviceClient.StartTerminal(options, args, workDir, interrupt, wait, restart, subcommand, stdout, stderr, stdin)
				}

				return exitError.Code, nil
			} else if restart {
				serviceClient.log.WriteString("\n")
				serviceClient.log.Infof("Restarting terminal because: %s", err)
				return serviceClient.StartTerminal(options, args, workDir, interrupt, wait, restart, subcommand, stdout, stderr, stdin)
			}

			return 0, err
		}
	}

	return 0, nil
}

func IsUnexpectedExitCode(code int) bool {
	// Expected exit codes are (https://shapeshed.com/unix-exit-codes/):
	// 1 - Catchall for general errors
	// 2 - Misuse of shell builtins (according to Bash documentation)
	// 126 - Command invoked cannot execute
	// 127 - “command not found”
	// 128 - Invalid argument to exit
	// 130 - Script terminated by Control-C
	return code != 0 && code != 1 && code != 2 && code != 126 && code != 127 && code != 128 && code != 130
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

// installScreen function check if screen command is present or not
// if not present then checks for distributor id of container
// and tries to install screen accordingly
func installScreen(container *selector.SelectedPodContainer, client kubectl.Client) (bool, error) {
	installScript := `
#!/bin/sh

which screen
if [ $? -eq 0 ]
then
	exit $?
else
	apt-get update && apt-get install screen -y
	if [ $? -eq 0 ]
	then
		exit $?
	else
		apk update && apk add screen
	fi
fi
exit $?
`
	cmd := []string{
		"sh",
		"-c",
		installScript,
	}
	_, _, err := client.ExecBuffered(container.Pod, container.Container.Name, cmd, nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

// getScreenCommand function checks if screen socket `dev` is present or not
// if not present then it creates a new socket `dev` and run the command from terminal.command config
// if socket `dev` is present then it reattaches it
func getScreenCommand(container *selector.SelectedPodContainer, client kubectl.Client, command []string) []string {
	stdout, _, err := client.ExecBuffered(container.Pod, container.Container.Name, []string{"screen", "-ls"}, nil)
	if err != nil {
		if strings.Contains(string(stdout), "No Sockets found") {
			cmd := []string{"screen", "-S", "dev"}
			cmd = append(cmd, command...)
			return cmd
		}
		return nil
	}
	return []string{"screen", "-x", "dev"}
}
