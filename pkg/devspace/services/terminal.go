package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"

	"github.com/mgutz/ansi"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartTerminal opens a new terminal
func (serviceClient *client) StartTerminal(options targetselector.Options, args []string, workDir string, interrupt chan error, wait bool) (int, error) {
	command := serviceClient.getCommand(args, workDir)
	targetSelector := targetselector.NewTargetSelector(serviceClient.client)
	if wait == false {
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
	go func() {
		interrupt <- serviceClient.client.ExecStreamWithTransport(&kubectl.ExecStreamWithTransportOptions{
			ExecStreamOptions: kubectl.ExecStreamOptions{
				Pod:       container.Pod,
				Container: container.Container.Name,
				Command:   command,
				TTY:       true,
				Stdin:     os.Stdin,
				Stdout:    os.Stdout,
				Stderr:    os.Stderr,
			},
			Transport:   wrapper,
			Upgrader:    upgradeRoundTripper,
			SubResource: kubectl.SubResourceExec,
		})
	}()

	err = <-interrupt
	upgradeRoundTripper.Close()
	if err != nil {
		if exitError, ok := err.(kubectlExec.CodeExitError); ok {
			return exitError.Code, nil
		}

		return 0, err
	}

	return 0, nil
}

func (serviceClient *client) getCommand(args []string, workDir string) []string {
	if serviceClient.config != nil && serviceClient.config.Config() != nil && serviceClient.config.Config().Dev.Terminal != nil {
		if len(args) == 0 {
			for _, cmd := range serviceClient.config.Config().Dev.Terminal.Command {
				args = append(args, cmd)
			}
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
