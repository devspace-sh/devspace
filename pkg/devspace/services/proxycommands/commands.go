package proxycommands

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

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
)

var DefaultRemotePort = 10567

// StartProxyCommands starts the reverse commands functionality
func StartProxyCommands(ctx devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	if ctx == nil || ctx.Config() == nil || ctx.Config().Config() == nil {
		return fmt.Errorf("DevSpace config is nil")
	}

	// init done array is used to track when sync was initialized
	initDoneArray := []chan struct{}{}
	remotePort := DefaultRemotePort
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		if len(devContainer.ProxyCommands) == 0 {
			return true
		}

		oldRemotePort := remotePort
		initDone := parent.NotifyGo(func() error {
			return startProxyCommands(ctx, devPod.Name, string(devContainer.Arch), oldRemotePort, devContainer.ProxyCommands, selector.WithContainer(devContainer.Container), parent)
		})
		initDoneArray = append(initDoneArray, initDone)
		remotePort++
		return true
	})

	// wait for init chans to be finished
	for _, initDone := range initDoneArray {
		<-initDone
	}
	return nil
}

func startProxyCommands(ctx devspacecontext.Context, name, arch string, remotePort int, reverseCommands []*latest.ProxyCommand, selector targetselector.TargetSelector, parent *tomb.Tomb) error {
	if ctx.IsDone() {
		return nil
	}

	// get a local port
	port, err := ssh.GetInstance(ctx.Log()).LockPort()
	if err != nil {
		return errors.Wrap(err, "find port")
	}

	defer ssh.GetInstance(ctx.Log()).ReleasePort(port)

	// start reverse port forwarding from that port
	mapping := fmt.Sprintf("%d:%d", port, remotePort)
	err = portforwarding.StartReversePortForwarding(ctx, name, arch, []*latest.PortMapping{
		{
			Port: mapping,
		},
	}, selector, parent)
	if err != nil {
		return errors.Wrap(err, "start ssh port forwarding")
	}

	// start ssh
	return startLocalSSH(ctx, selector, reverseCommands, remotePort, fmt.Sprintf(":%d", port), parent)
}

func startLocalSSH(ctx devspacecontext.Context, selector targetselector.TargetSelector, reverseCommands []*latest.ProxyCommand, remotePort int, addr string, parent *tomb.Tomb) error {
	if ctx.IsDone() {
		return nil
	}

	// find target container
	container, err := selector.SelectSingleContainer(ctx.Context(), ctx.KubeClient(), ctx.Log())
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
	gitCredentials := false
	for _, r := range reverseCommands {
		if r.GitCredentials {
			gitCredentials = true
		}
		if r.Command == "" {
			continue
		}

		commandsToReplace = append(commandsToReplace, r.Command)
	}

	// execute configure command in container
	command := []string{inject.DevSpaceHelperContainerPath, "proxy-commands", "configure", "--port", strconv.Itoa(remotePort), "--public-key", base64.StdEncoding.EncodeToString([]byte(publicKey)), "--private-key", base64.StdEncoding.EncodeToString([]byte(privateKey))}
	if len(commandsToReplace) > 0 {
		command = append(command, "--commands", strings.Join(commandsToReplace, ","))
	}
	if gitCredentials {
		command = append(command, "--git-credentials")
	}

	stdout, stderr, err := ctx.KubeClient().ExecBuffered(ctx.Context(), container.Pod, container.Container.Name, command, nil)
	if err != nil {
		return fmt.Errorf("error setting up proxy commands in container: %s %s %v", string(stdout), string(stderr), err)
	}
	containerWorkingDir := strings.TrimSpace(string(stdout))
	if containerWorkingDir == "" {
		return fmt.Errorf("couldn't retrieve container working dir")
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
	sshServer := NewReverseCommandsServer(ctx.WorkingDir(), containerWorkingDir, addr, keys, reverseCommands, ctx.Log())
	parent.Go(func() error {
		return sshServer.ListenAndServe(ctx.Context())
	})
	return nil
}
