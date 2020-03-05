package services

import (
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"github.com/mgutz/ansi"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartTerminal opens a new terminal
func (serviceClient *client) StartTerminal(args []string, imageSelector []string, interrupt chan error, wait bool) (int, error) {
	command := serviceClient.getCommand(args)

	targetSelector, err := targetselector.NewTargetSelector(serviceClient.config, serviceClient.client, serviceClient.selectorParameter, true, imageSelector)
	if err != nil {
		return 0, err
	}

	// Allow picking non running pods
	targetSelector.AllowNonRunning = true
	targetSelector.SkipWait = !wait
	targetSelector.PodQuestion = ptr.String("Which pod do you want to open the terminal for?")

	pod, container, err := targetSelector.GetContainer(true, serviceClient.log)
	if err != nil {
		return 0, err
	}

	wrapper, upgradeRoundTripper, err := serviceClient.client.GetUpgraderWrapper()
	if err != nil {
		return 0, err
	}

	serviceClient.log.Infof("Opening shell to pod:container %s:%s", ansi.Color(pod.Name, "white+b"), ansi.Color(container.Name, "white+b"))
	if serviceClient.selectorParameter.CmdParameter.Interactive == true && len(container.Command) > 0 && serviceClient.config != nil && serviceClient.generated != nil && serviceClient.config.Dev != nil && serviceClient.config.Dev.Interactive != nil && len(serviceClient.config.Dev.Interactive.Images) > 0 {
		for _, image := range serviceClient.config.Dev.Interactive.Images {
			imageSelector = []string{}
			imageConfigCache := serviceClient.generated.GetActive().GetImageCache(image.Name)
			if imageConfigCache != nil && imageConfigCache.ImageName != "" {
				imageName := imageConfigCache.ImageName + ":" + imageConfigCache.Tag
				if imageName == container.Image && (len(image.Entrypoint) > 0 || len(image.Cmd) > 0) {
					serviceClient.log.Warnf("The container you are entering was started with a Kubernetes `command` option (%s) instead of the original Dockerfile ENTRYPOINT. Interactive mode ENTRYPOINT override does not work for containers started using with Kubernetes command.", container.Command)
				}
			}
		}
	}

	go func() {
		interrupt <- serviceClient.client.ExecStreamWithTransport(wrapper, upgradeRoundTripper, pod, container.Name, command, true, os.Stdin, os.Stdout, os.Stderr, kubectl.SubResourceExec)
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

func (serviceClient *client) getCommand(args []string) []string {
	var command []string

	config := serviceClient.config
	if config != nil && config.Dev != nil && config.Dev.Interactive != nil && config.Dev.Interactive.Terminal != nil && len(config.Dev.Interactive.Terminal.Command) > 0 {
		for _, cmd := range config.Dev.Interactive.Terminal.Command {
			command = append(command, cmd)
		}
	}

	if len(args) > 0 {
		command = args
	} else {
		if len(command) == 0 {
			command = []string{
				"sh",
				"-c",
				"command -v bash >/dev/null 2>&1 && exec bash || exec sh",
			}
		}
	}

	return command
}
