package hook

import (
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"io"
)

func NewRemoteCommandHook(stdout io.Writer, stderr io.Writer) RemoteHook {
	return &remoteCommandHook{
		Stdout: stdout,
		Stderr: stderr,
	}
}

type remoteCommandHook struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (r *remoteCommandHook) ExecuteRemotely(hook *latest.HookConfig, podContainer *selector.SelectedPodContainer, client kubectl.Client, config config.Config, dependencies []types.Dependency, log logpkg.Logger) error {
	hookCommand, hookArgs, err := resolveCommand(hook.Command, hook.Args, config, dependencies)
	if err != nil {
		return err
	}

	cmd := []string{hookCommand}
	if hook.Args == nil {
		cmd = []string{"sh", "-c", hookCommand}
	} else {
		cmd = append(cmd, hookArgs...)
	}

	err = client.ExecStream(&kubectl.ExecStreamOptions{
		Pod:       podContainer.Pod,
		Container: podContainer.Container.Name,
		Command:   cmd,
		Stdout:    r.Stdout,
		Stderr:    r.Stderr,
	})
	if err != nil {
		return errors.Errorf("error in container '%s/%s/%s': %v", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
	}

	return nil
}
