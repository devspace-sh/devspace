package hook

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
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

func (r *remoteCommandHook) ExecuteRemotely(ctx Context, hook *latest.HookConfig, podContainer *kubectl.SelectedPodContainer, log logpkg.Logger) error {
	cmd := []string{hook.Command}
	cmd = append(cmd, hook.Args...)
	err := ctx.Client.ExecStream(&kubectl.ExecStreamOptions{
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
