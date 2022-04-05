package hook

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"io"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/mgutz/ansi"
)

func NewLogsHook(writer io.Writer) RemoteHook {
	return &remoteLogsHook{
		Writer: writer,
	}
}

type remoteLogsHook struct {
	Writer io.Writer
}

func (r *remoteLogsHook) ExecuteRemotely(ctx devspacecontext.Context, hook *latest.HookConfig, podContainer *selector.SelectedPodContainer) error {
	ctx.Log().Infof("Execute hook '%s' in container '%s/%s/%s'", ansi.Color(hookName(hook), "white+b"), podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name)
	reader, err := ctx.KubeClient().Logs(ctx.Context(), podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, false, hook.Logs.TailLines, true)
	if err != nil {
		return err
	}

	_, err = io.Copy(r.Writer, reader)
	return err
}
