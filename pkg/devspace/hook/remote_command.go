package hook

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/hash"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/exec"
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
	hookCommand, hookArgs, err := ResolveCommand(hook.Command, hook.Args, config, dependencies)
	if err != nil {
		return err
	}

	cmd := []string{hookCommand}
	if hook.Args == nil {
		cmd = []string{"sh", "-c", hookCommand}
	} else {
		cmd = append(cmd, hookArgs...)
	}

	once := hook.Container.Once != nil && *hook.Container.Once
	if once {
		// check whether hook has previously executed
		hookExecuted, err := hasHookExecuted(hookCommand, hookArgs, podContainer, client)
		if err != nil {
			return errors.Errorf("error checking whether hook has executed '%s/%s/%s': %v", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
		}

		if hookExecuted {
			log.Infof("Skip hook '%s' because it is configured to run once", ansi.Color(hookName(hook), "white+b"))
			return nil
		}
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

	log.Infof("Execute hook '%s' in container '%s/%s/%s'", ansi.Color(hookName(hook), "white+b"), podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name)
	err = client.ExecStream(&kubectl.ExecStreamOptions{
		Pod:       podContainer.Pod,
		Container: podContainer.Container.Name,
		Command:   cmd,
		Stdout:    io.MultiWriter(r.Stdout, stdout),
		Stderr:    io.MultiWriter(r.Stderr, stderr),
	})
	if err != nil {
		return errors.Errorf("error in container '%s/%s/%s': %v", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
	}

	if once {
		// record hook execution
		err := recordHookExecuted(hookCommand, hookArgs, podContainer, client)
		if err != nil {
			return errors.Errorf("error recording hook execution %s in container '%s/%s/%s': %v", ansi.Color(hookName(hook), "white+b"), podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
		}
	}

	return nil
}

func commandHash(command string, args []string) string {
	return hash.String(fmt.Sprintf("%s %s", command, strings.Join(args, " ")))
}

func hasHookExecuted(command string, args []string, podContainer *selector.SelectedPodContainer, client kubectl.Client) (bool, error) {
	cmdHash := commandHash(command, args)
	cmd := []string{"test", "-e", fmt.Sprintf(`/tmp/hook-%s`, cmdHash)}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := client.ExecStream(&kubectl.ExecStreamOptions{
		Pod:       podContainer.Pod,
		Container: podContainer.Container.Name,
		Command:   cmd,
		Stdout:    stdout,
		Stderr:    stderr,
	})
	if err != nil {
		if errors.As(err, &exec.CodeExitError{}) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func recordHookExecuted(command string, args []string, podContainer *selector.SelectedPodContainer, client kubectl.Client) error {
	cmdHash := commandHash(command, args)
	cmd := []string{"touch", fmt.Sprintf(`/tmp/hook-%s`, cmdHash)}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := client.ExecStream(&kubectl.ExecStreamOptions{
		Pod:       podContainer.Pod,
		Container: podContainer.Container.Name,
		Command:   cmd,
		Stdout:    stdout,
		Stderr:    stderr,
	})
	if err != nil {
		return err
	}

	return nil
}
