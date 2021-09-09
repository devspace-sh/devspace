package hook

import (
	"encoding/json"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/util"
	"github.com/loft-sh/devspace/pkg/util/command"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/shell"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
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
	dir := filepath.Dir(config.Path())

	// resolve hook command and args
	hookCommand, hookArgs, err := resolveCommand(hook.Command, hook.Args, config, dependencies)
	if err != nil {
		return err
	}

	// if args are nil we execute the command in a shell
	if hook.Args == nil {
		return shell.ExecuteShellCommand(hookCommand, nil, dir, l.Stdout, l.Stderr, extraEnv)
	}

	// else we execute it directly
	return command.ExecuteCommandWithEnv(hookCommand, hookArgs, dir, l.Stdout, l.Stderr, extraEnv)
}

func resolveCommand(command string, args []string, config config.Config, dependencies []types.Dependency) (string, []string, error) {
	// resolve hook command
	hookCommand, err := util.ResolveImageHelpers(command, config, dependencies)
	if err != nil {
		return "", nil, errors.Wrap(err, "resolve image helpers")
	}

	// resolve args
	if args != nil {
		newArgs := []string{}
		for _, a := range args {
			newArg, err := util.ResolveImageHelpers(a, config, dependencies)
			if err != nil {
				return "", nil, errors.Wrap(err, "resolve image helpers")
			}

			newArgs = append(newArgs, newArg)
		}

		return hookCommand, newArgs, nil
	}

	return hookCommand, nil, nil
}
