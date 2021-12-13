package hook

import (
	"bytes"
	"encoding/json"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"io"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/command"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/shell"
	"github.com/pkg/errors"
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

func (l *localCommandHook) Execute(hook *latest.HookConfig, client kubectl.Client, config config.Config, dependencies []types.Dependency, cmdExtraEnv map[string]string, log logpkg.Logger) error {
	// Create extra env variables
	osArgsBytes, err := json.Marshal(os.Args)
	if err != nil {
		return err
	}
	extraEnv := map[string]string{
		OsArgsEnv: string(osArgsBytes),
	}
	if client != nil {
		extraEnv[KubeContextEnv] = client.CurrentContext()
		extraEnv[KubeNamespaceEnv] = client.Namespace()
	}
	for k, v := range cmdExtraEnv {
		extraEnv[k] = v
	}

	dir := filepath.Dir(config.Path())

	// resolve hook command and args
	hookCommand, hookArgs, err := ResolveCommand(hook.Command, hook.Args, config, dependencies)
	if err != nil {
		return err
	}

	// if args are nil we execute the command in a shell
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	defer func() {
		if hook.Name != "" {
			config.SetRuntimeVariable("hooks."+hook.Name+".stdout", stdout.String())
			config.SetRuntimeVariable("hooks."+hook.Name+".stderr", stderr.String())
		}
	}()

	if hook.Args == nil {
		return shell.ExecuteShellCommand(hookCommand, nil, dir, io.MultiWriter(l.Stdout, stdout), io.MultiWriter(l.Stderr, stderr), extraEnv)
	}

	// else we execute it directly
	return command.ExecuteCommandWithEnv(hookCommand, hookArgs, dir, io.MultiWriter(l.Stdout, stdout), io.MultiWriter(l.Stderr, stderr), extraEnv)
}

func ResolveCommand(command string, args []string, config config.Config, dependencies []types.Dependency) (string, []string, error) {
	// resolve hook command
	hookCommand, err := runtimevar.NewRuntimeResolver(true).FillRuntimeVariablesAsString(command, config, dependencies)
	if err != nil {
		return "", nil, errors.Wrap(err, "resolve image helpers")
	}

	// resolve args
	if args != nil {
		newArgs := []string{}
		for _, a := range args {
			newArg, err := runtimevar.NewRuntimeResolver(true).FillRuntimeVariablesAsString(a, config, dependencies)
			if err != nil {
				return "", nil, errors.Wrap(err, "resolve image helpers")
			}

			newArgs = append(newArgs, newArg)
		}

		return hookCommand, newArgs, nil
	}

	return hookCommand, nil, nil
}
