package hook

import (
	"encoding/json"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/command"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"io"
	"os"
)

func NewLocalCommandHook(stdout io.Writer, stderr io.Writer) Hook {
	return &localCommandHook{
		Stdout: stdout,
		Stderr: stderr,
	}
}

type localCommandHook struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (l *localCommandHook) Execute(ctx Context, hook *latest.HookConfig, log logpkg.Logger) error {
	// Create extra env variables
	osArgsBytes, err := json.Marshal(os.Args)
	if err != nil {
		return err
	}
	extraEnv := map[string]string{
		OsArgsEnv: string(osArgsBytes),
	}
	if ctx.Client != nil {
		extraEnv[KubeContextEnv] = ctx.Client.CurrentContext()
		extraEnv[KubeNamespaceEnv] = ctx.Client.Namespace()
	}
	if ctx.Error != nil {
		extraEnv[ErrorEnv] = ctx.Error.Error()
	}

	err = command.ExecuteCommandWithEnv(hook.Command, hook.Args, l.Stdout, l.Stderr, extraEnv)
	if err != nil {
		return err
	}

	return nil
}
