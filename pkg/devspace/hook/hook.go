package hook

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/command"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	dockerterm "github.com/moby/term"
	"github.com/pkg/errors"
	"io"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
	"time"
)

var (
	_, stdout, stderr = dockerterm.StdStreams()
)

const (
	KubeContextEnv   = "DEVSPACE_HOOK_KUBE_CONTEXT"
	KubeNamespaceEnv = "DEVSPACE_HOOK_KUBE_NAMESPACE"
	OsArgsEnv        = "DEVSPACE_HOOK_OS_ARGS"
)

type Events []string

func (e Events) With(name string) Events {
	return append(e, name)
}

func EventsForSingle(base, name string) Events {
	if name == "" {
		return []string{base + ":*"}
	}

	return []string{base + ":*", base + ":" + name}
}

// Hook is an interface to execute a specific hook type
type Hook interface {
	Execute(hook *latest.HookConfig, client kubectl.Client, config config.Config, dependencies []types.Dependency, extraEnv map[string]string, log logpkg.Logger) error
}

// LogExecuteHooks executes plugin hooks and config hooks and prints errors to the log
func LogExecuteHooks(client kubectl.Client, config config.Config, dependencies []types.Dependency, extraEnv map[string]interface{}, log logpkg.Logger, events ...string) {
	// call plugin first
	plugin.LogExecutePluginHookWithContext(extraEnv, events...)

	// now execute hooks
	if config != nil {
		if log == nil {
			log = logpkg.GetInstance()
		}

		convertedExtraEnv := plugin.ConvertExtraEnv("DEVSPACE_HOOK", extraEnv)
		for _, e := range events {
			convertedExtraEnv["DEVSPACE_HOOK_EVENT"] = e
			err := executeSingle(client, config, dependencies, convertedExtraEnv, log, e)
			if err != nil {
				log.Warn(err)
			}
		}
	}
}

// ExecuteHooks executes plugin hooks and config hooks
func ExecuteHooks(client kubectl.Client, config config.Config, dependencies []types.Dependency, extraEnv map[string]interface{}, log logpkg.Logger, events ...string) error {
	// call plugin first
	err := plugin.ExecutePluginHookWithContext(extraEnv, events...)
	if err != nil {
		return err
	}

	// now execute hooks
	if config != nil {
		if log == nil {
			log = logpkg.GetInstance()
		}

		convertedExtraEnv := plugin.ConvertExtraEnv("DEVSPACE_HOOK", extraEnv)
		for _, e := range events {
			convertedExtraEnv["DEVSPACE_HOOK_EVENT"] = e
			err := executeSingle(client, config, dependencies, convertedExtraEnv, log, e)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// executeSingle executes hooks at a specific time
func executeSingle(client kubectl.Client, config config.Config, dependencies []types.Dependency, extraEnv map[string]string, log logpkg.Logger, event string) error {
	if config == nil {
		return nil
	}

	c := config.Config()
	if c.Hooks != nil && len(c.Hooks) > 0 {
		hooksToExecute := []*latest.HookConfig{}

		// Gather all hooks we should execute
		for _, hook := range c.Hooks {
			for _, e := range hook.Events {
				if e == event {
					hooksToExecute = append(hooksToExecute, hook)
					break
				}
			}
		}

		// Execute hooks
		for _, hookConfig := range hooksToExecute {
			if command.ShouldExecuteOnOS(hookConfig.OperatingSystem) == false {
				continue
			}

			// Determine output writer
			var writer io.Writer
			if log == logpkg.GetInstance() {
				writer = stdout
			} else {
				writer = log
			}

			// If the hook is silent, we cache it in a buffer
			hookWriter := writer
			if hookConfig.Silent {
				hookWriter = &bytes.Buffer{}
			}

			// Decide which hook type to use
			var hook Hook
			if hookConfig.Container != nil {
				if hookConfig.Upload != nil {
					hook = NewRemoteHook(NewUploadHook())
				} else if hookConfig.Download != nil {
					hook = NewRemoteHook(NewDownloadHook())
				} else if hookConfig.Logs != nil {
					// we use another waiting strategy here, because the pod might has finished already
					hook = NewRemoteHookWithWaitingStrategy(NewLogsHook(hookWriter), targetselector.NewUntilNotWaitingStrategy(time.Second*2))
				} else if hookConfig.Wait != nil {
					hook = NewWaitHook()
				} else {
					hook = NewRemoteHook(NewRemoteCommandHook(hookWriter, hookWriter))
				}
			} else {
				hook = NewLocalCommandHook(hookWriter, hookWriter)
			}

			// Execute the hook
			err := executeHook(hookConfig, hookWriter, client, config, dependencies, extraEnv, log, hook, event)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func executeHook(hookConfig *latest.HookConfig, hookWriter io.Writer, client kubectl.Client, config config.Config, dependencies []types.Dependency, extraEnv map[string]string, log logpkg.Logger, hook Hook, event string) error {
	hookLog := log
	if hookConfig.Silent {
		hookLog = logpkg.Discard
	}

	if hookConfig.Background {
		log.Infof("Execute hook '%s' in background at %s", ansi.Color(hookName(hookConfig), "white+b"), ansi.Color(event, "white+b"))
		go func() {
			err := hook.Execute(hookConfig, client, config, dependencies, extraEnv, hookLog)
			if err != nil {
				if hookConfig.Silent {
					log.Warnf("Error executing hook '%s' in background: %s %v", ansi.Color(hookName(hookConfig), "white+b"), hookWriter.(*bytes.Buffer).String(), err)
				} else {
					log.Warnf("Error executing hook '%s' in background: %v", ansi.Color(hookName(hookConfig), "white+b"), err)
				}
			}
		}()

		return nil
	}

	log.Infof("Execute hook '%s' at %s", ansi.Color(hookName(hookConfig), "white+b"), ansi.Color(event, "white+b"))
	err := hook.Execute(hookConfig, client, config, dependencies, extraEnv, hookLog)
	if err != nil {
		if hookConfig.Silent {
			return errors.Wrapf(err, "in hook '%s': %s", ansi.Color(hookName(hookConfig), "white+b"), hookWriter.(*bytes.Buffer).String())
		} else {
			return errors.Wrapf(err, "in hook '%s'", ansi.Color(hookName(hookConfig), "white+b"))
		}
	}

	return nil
}

func hookName(hook *latest.HookConfig) string {
	if hook.Command != "" {
		commandString := strings.TrimSpace(hook.Command + " " + strings.Join(hook.Args, " "))
		splitted := strings.Split(commandString, "\n")
		if len(splitted) > 1 {
			return splitted[0] + "..."
		}

		return commandString
	}
	if hook.Upload != nil && hook.Container != nil {
		localPath := "."
		if hook.Upload.LocalPath != "" {
			localPath = hook.Upload.LocalPath
		}
		containerPath := "."
		if hook.Upload.ContainerPath != "" {
			containerPath = hook.Upload.ContainerPath
		}

		if hook.Container.Pod != "" {
			return fmt.Sprintf("copy %s to pod %s", localPath, hook.Container.Pod)
		}
		if len(hook.Container.LabelSelector) > 0 {
			return fmt.Sprintf("copy %s to selector %s", localPath, labels.Set(hook.Container.LabelSelector).String())
		}
		if hook.Container.ImageSelector != "" {
			return fmt.Sprintf("copy %s to image %s", localPath, hook.Container.ImageSelector)
		}

		return fmt.Sprintf("copy %s to %s", localPath, containerPath)
	}
	if hook.Download != nil && hook.Container != nil {
		localPath := "."
		if hook.Download.LocalPath != "" {
			localPath = hook.Download.LocalPath
		}
		containerPath := "."
		if hook.Download.ContainerPath != "" {
			containerPath = hook.Download.ContainerPath
		}

		if hook.Container.Pod != "" {
			return fmt.Sprintf("download from pod %s to %s", hook.Container.Pod, localPath)
		}
		if len(hook.Container.LabelSelector) > 0 {
			return fmt.Sprintf("download from selector %s to %s", labels.Set(hook.Container.LabelSelector).String(), localPath)
		}
		if hook.Container.ImageSelector != "" {
			return fmt.Sprintf("download from image %s to %s", hook.Container.ImageSelector, localPath)
		}

		return fmt.Sprintf("download from container:%s to local:%s", containerPath, localPath)
	}
	if hook.Logs != nil && hook.Container != nil {
		if hook.Container.Pod != "" {
			return fmt.Sprintf("logs from pod %s", hook.Container.Pod)
		}
		if len(hook.Container.LabelSelector) > 0 {
			return fmt.Sprintf("logs from selector %s", labels.Set(hook.Container.LabelSelector).String())
		}
		if hook.Container.ImageSelector != "" {
			return fmt.Sprintf("logs from image %s", hook.Container.ImageSelector)
		}

		return fmt.Sprintf("logs from first container found")
	}
	if hook.Wait != nil && hook.Container != nil {
		if hook.Container.Pod != "" {
			return fmt.Sprintf("wait for pod %s", hook.Container.Pod)
		}
		if len(hook.Container.LabelSelector) > 0 {
			return fmt.Sprintf("wait for selector %s", labels.Set(hook.Container.LabelSelector).String())
		}
		if hook.Container.ImageSelector != "" {
			return fmt.Sprintf("wait for image %s", hook.Container.ImageSelector)
		}

		return fmt.Sprintf("wait for everything")
	}
	return "hook"
}
