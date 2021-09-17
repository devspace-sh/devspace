package hook

import (
	"context"
	"io"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
)

func NewLogsHook(writer io.Writer) RemoteHook {
	return &remoteLogsHook{
		Writer: writer,
	}
}

type remoteLogsHook struct {
	Writer io.Writer
}

func (r *remoteLogsHook) ExecuteRemotely(ctx Context, hook *latest.HookConfig, podContainer *selector.SelectedPodContainer, config config.Config, dependencies []types.Dependency, log logpkg.Logger) error {
	reader, err := ctx.Client.Logs(context.TODO(), podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, false, hook.Logs.TailLines, true)
	if err != nil {
		return err
	}

	_, err = io.Copy(r.Writer, reader)
	return err
}
