package reversecommands

import (
	"encoding/base64"
	"fmt"
	sshpkg "github.com/gliderlabs/ssh"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/devspace/services/portforwarding"
	"github.com/loft-sh/devspace/pkg/devspace/services/ssh"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/pkg/errors"
	"strings"
)

var DefaultRemotePort = 10567

// StartReverseCommands starts the reverse commands functionality
func StartReverseCommands(ctx *devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, parent *tomb.Tomb) (retErr error) {
	if ctx == nil || ctx.Config == nil || ctx.Config.Config() == nil {
		return fmt.Errorf("DevSpace config is nil")
	}

	// init done array is used to track when sync was initialized
	initDoneArray := []chan struct{}{}
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		if len(devContainer.ReverseCommands) == 0 {
			return true
		}

		initDone := parent.NotifyGo(func() error {
			return startReverseCommands(ctx, devPod.Name, string(devContainer.Arch), devContainer.ReverseCommands, selector.WithContainer(devContainer.Container), parent)
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

func startReverseCommands(ctx *devspacecontext.Context, name, arch string, reverseCommands []*latest.ReverseCommand, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	if ctx.IsDone() {
		return nil
	}

	// get a local port
	port, err := ssh.LockPort()
	if err != nil {
		return errors.Wrap(err, "find port")
	}

	defer ssh.ReleasePort(port)

	// get remote port
	defaultRemotePort := DefaultRemotePort

	// start reverse port forwarding from that port
	mapping := fmt.Sprintf("%d:%d", port, defaultRemotePort)
	err = portforwarding.StartReversePortForwarding(ctx, name, arch, []*latest.PortMapping{
		{
			Port: mapping,
		},
	}, selector, parent)
	if err != nil {
		return errors.Wrap(err, "start ssh port forwarding")
	}

	// start ssh
	return startLocalSSH(ctx, selector, reverseCommands, fmt.Sprintf(":%d", port), parent)
}

func startLocalSSH(ctx *devspacecontext.Context, selector targetselector.TargetSelector, reverseCommands []*latest.ReverseCommand, addr string, parent *tomb.Tomb) error {
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

	// create a new public / private key
	publicKey, privateKey, err := ssh.MakeSSHKeyPair()
	if err != nil {
		return errors.Wrap(err, "generate key pair")
	}

	// gather all commands that should get replaced in the container
	commandsToReplace := []string{}
	for _, r := range reverseCommands {
		if r.Name == "" {
			continue
		}

		commandsToReplace = append(commandsToReplace, r.Name)
	}

	// execute configure command in container
	command := []string{inject.DevSpaceHelperContainerPath, "reverse-commands", "configure", "--public-key", base64.StdEncoding.EncodeToString([]byte(publicKey)), "--private-key", base64.StdEncoding.EncodeToString([]byte(privateKey)), "--commands", strings.Join(commandsToReplace, ",")}
	stdout, err := ctx.KubeClient.ExecBufferedCombined(ctx.Context, container.Pod, container.Container.Name, command, nil)
	if err != nil {
		return fmt.Errorf("error setting up reverse commands in container: %s %v", string(stdout), err)
	}

	// parse key
	var keys []sshpkg.PublicKey
	keyBytes := []byte(publicKey)
	for len(keyBytes) > 0 {
		key, _, _, rest, err := sshpkg.ParseAuthorizedKey(keyBytes)
		if err != nil {
			return errors.Wrap(err, "parse authorized key")
		}

		keys = append(keys, key)
		keyBytes = rest
	}

	// start local ssh server
	sshServer := NewReverseCommandsServer(ctx.WorkingDir, addr, keys, reverseCommands, ctx.Log)
	parent.Go(func() error {
		return sshServer.ListenAndServe(ctx.Context)
	})
	return nil
}
