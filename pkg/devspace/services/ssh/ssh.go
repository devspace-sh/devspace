package ssh

import (
	"bytes"
	"fmt"
	"github.com/mgutz/ansi"
	"io"
	"strconv"
	"strings"
	"time"

	helperssh "github.com/loft-sh/devspace/helper/ssh"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/devspace/services/portforwarding"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/devspace/services/terminal"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kubectlExec "k8s.io/client-go/util/exec"
)

// StartSSH starts the ssh functionality
func StartSSH(ctx *devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, parent *tomb.Tomb) (retErr error) {
	if ctx == nil || ctx.Config == nil || ctx.Config.Config() == nil {
		return fmt.Errorf("DevSpace config is nil")
	}

	// init done array is used to track when sync was initialized
	initDoneArray := []chan struct{}{}
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		if devContainer.SSH == nil {
			return true
		}

		initDone := parent.NotifyGo(func() error {
			return startSSH(ctx, devPod.Name, string(devContainer.Arch), devContainer.SSH, selector.WithContainer(devContainer.Container), parent)
		})
		initDoneArray = append(initDoneArray, initDone)
		return true
	})

	// wait for init chans to be finished
	for _, initDone := range initDoneArray {
		<-initDone
	}
	return nil
}

func startSSH(ctx *devspacecontext.Context, name, arch string, sshConfig *latest.SSH, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	if ctx.IsDone() {
		return nil
	}

	// get a local port
	port := sshConfig.LocalPort
	var err error
	if port == 0 {
		port, err = LockPort()
		if err != nil {
			return errors.Wrap(err, "find port")
		}

		defer ReleasePort(port)
	}

	// configure ssh host
	sshHost := name + "." + ctx.Config.Config().Name + ".devspace"
	if sshConfig.LocalHostname != "" {
		sshHost = sshConfig.LocalHostname
	}

	// get remote port
	defaultRemotePort := helperssh.DefaultPort
	if sshConfig.RemoteAddress != "" {
		splitted := strings.Split(sshConfig.RemoteAddress, ":")
		if len(splitted) != 2 {
			return fmt.Errorf("invalid ssh address %s, must contain host:port", sshConfig.RemoteAddress)
		}

		defaultRemotePort, err = strconv.Atoi(splitted[1])
		if err != nil {
			return fmt.Errorf("error parsing remote port %s: %v", splitted[1], err)
		}
	}

	// start port forwarding to that port
	mapping := fmt.Sprintf("%d:%d", port, defaultRemotePort)
	err = portforwarding.StartForwarding(ctx, name, []*latest.PortMapping{
		{
			Port: mapping,
		},
	}, selector, parent)
	if err != nil {
		return errors.Wrap(err, "start ssh port forwarding")
	}

	// update ssh config
	err = configureSSHConfig(sshHost, strconv.Itoa(port), ctx.Log)
	if err != nil {
		return errors.Wrap(err, "update ssh config")
	}

	// start ssh
	return startSSHWithRestart(ctx, arch, sshConfig.RemoteAddress, sshHost, selector, parent)
}

func startSSHWithRestart(ctx *devspacecontext.Context, arch, addr, sshHost string, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	if ctx.IsDone() {
		return nil
	}

	// find target container
	container, err := selector.SelectSingleContainer(ctx.Context, ctx.KubeClient, ctx.Log)
	if err != nil {
		return errors.Wrap(err, "error selecting container")
	} else if container == nil {
		return nil
	}

	// make sure the DevSpace helper binary is injected
	err = inject.InjectDevSpaceHelper(ctx.Context, ctx.KubeClient, container.Pod, container.Container.Name, arch, ctx.Log)
	if err != nil {
		return err
	}

	// get public key
	publicKey, err := getPublicKey()
	if err != nil {
		return errors.Wrap(err, "generate key pair")
	}

	// get command
	command := []string{inject.DevSpaceHelperContainerPath, "ssh", "--authorized-key", publicKey}
	if addr != "" {
		command = append(command, "--address", addr)
	}

	// start ssh server
	parent.Go(func() error {
		writer := ctx.Log.Writer(logrus.DebugLevel, false)
		defer writer.Close()
		for !ctx.IsDone() {
			buffer := &bytes.Buffer{}
			multiWriter := io.MultiWriter(writer, buffer)
			err = ctx.KubeClient.ExecStream(ctx.Context, &kubectl.ExecStreamOptions{
				Pod:         container.Pod,
				Container:   container.Container.Name,
				Command:     command,
				Stdout:      multiWriter,
				Stderr:      multiWriter,
				SubResource: kubectl.SubResourceExec,
			})
			if err != nil {
				select {
				case <-ctx.Context.Done():
					return nil
				case <-time.After(time.Second * 2):
					// check if context is done
					if exitError, ok := err.(kubectlExec.CodeExitError); ok {
						if terminal.IsUnexpectedExitCode(exitError.Code) {
							ctx.Log.Warnf("restarting ssh process because: %s %v", buffer.String(), err)
							continue
						}

						return fmt.Errorf("ssh server failed: %s %v", buffer.String(), err)
					}

					ctx.Log.Warnf("restarting ssh process because: %s %v", buffer.String(), err)
					continue
				}
			}

			// seems like the ssh process is still running
			<-ctx.Context.Done()
			return nil
		}

		return nil
	})

	ctx.Log.Donef("Use '%s' to connect via SSH", ansi.Color(fmt.Sprintf("ssh %s", sshHost), "white+b"))
	return nil
}
