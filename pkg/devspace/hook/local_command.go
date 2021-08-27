package hook

import (
	"encoding/json"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/util/command"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/shell"
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

func (l *localCommandHook) Execute(ctx Context, hook *latest.HookConfig, config config.Config, dependencies []types.Dependency, log logpkg.Logger) error {
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

	// if args are nil we execute the command in a shell
	if hook.Args == nil {
		return shell.ExecuteShellCommand(hook.Command, nil, l.Stdout, l.Stderr, extraEnv)
	}

	// else we execute it directly
	return command.ExecuteCommandWithEnv(hook.Command, hook.Args, l.Stdout, l.Stderr, extraEnv)
}
