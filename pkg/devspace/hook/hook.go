package hook

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/command"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	dockerterm "github.com/moby/term"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	KubeContextEnv   = "DEVSPACE_HOOK_KUBE_CONTEXT"
	KubeNamespaceEnv = "DEVSPACE_HOOK_KUBE_NAMESPACE"
	ErrorEnv         = "DEVSPACE_HOOK_ERROR"
	OsArgsEnv        = "DEVSPACE_HOOK_OS_ARGS"
)

// Hook is an interface to execute a specific hook type
type Hook interface {
	Execute(ctx Context, hook *latest.HookConfig, config config.Config, dependencies []types.Dependency, log logpkg.Logger) error
}

// Executer executes configured commands locally
type Executer interface {
	OnError(stage Stage, whichs []string, context Context, log logpkg.Logger)
	Execute(when When, stage Stage, which string, context Context, log logpkg.Logger) error
	ExecuteMultiple(when When, stage Stage, whichs []string, context Context, log logpkg.Logger) error
}

type executer struct {
	config       config.Config
	dependencies []types.Dependency
}

// NewExecuter creates an instance of Executer for the specified config
func NewExecuter(config config.Config, dependencies []types.Dependency) Executer {
	return &executer{
		config:       config,
		dependencies: dependencies,
	}
}

// When is the type that is used to tell devspace when relatively to a stage a hook should be executed
type When string

const (
	// Before is used to tell devspace to execute a hook before a certain stage
	Before When = "before"
	// After is used to tell devspace to execute a hook after a certain stage
	After When = "after"
	// OnError is used to tell devspace to execute a hook after a certain error occurred
	OnError When = "onError"
)

// Stage is the type that defines the stage at when to execute a hook
type Stage string

const (
	// StageImages is the image building stage
	StageImages Stage = "images"
	// StageDeployments is the deploying stage
	StageDeployments Stage = "deployments"
	// StagePurgeDeployments is the purging stage
	StagePurgeDeployments Stage = "purgeDeployments"
	// StageDependencies is the dependency stage
	StageDependencies Stage = "dependencies"
	// StagePullSecrets is the pull secrets stage
	StagePullSecrets Stage = "pullSecrets"
	// StageInitialSync is the initial sync stage
	StageInitialSync Stage = "initialSync"
)

// All is used to tell devspace to execute a hook before or after all images, deployments
const All = "all"

var (
	_, stdout, _ = dockerterm.StdStreams()
)

// Context holds hook context information
type Context struct {
	Error  error
	Client kubectl.Client
}

// ExecuteMultiple executes multiple hooks at a specific time
func (e *executer) ExecuteMultiple(when When, stage Stage, whichs []string, context Context, log logpkg.Logger) error {
	for _, which := range whichs {
		err := e.Execute(when, stage, which, context, log)
		if err != nil {
			return err
		}
	}

	return nil
}

// OnError is a convenient method to handle the resulting error of a hook execution.
// Since we mostly return anyways after an error has occurred this only prints additional information why the hook failed
func (e *executer) OnError(stage Stage, whichs []string, context Context, log logpkg.Logger) {
	err := e.ExecuteMultiple(OnError, stage, whichs, context, log)
	if err != nil {
		log.Warnf("Hook failed: %v", err)
	}
}

// Execute executes hooks at a specific time
func (e *executer) Execute(when When, stage Stage, which string, context Context, log logpkg.Logger) error {
	if e.config == nil {
		return nil
	}

	c := e.config.Config()
	if c.Hooks != nil && len(c.Hooks) > 0 {
		hooksToExecute := []*latest.HookConfig{}

		// Gather all hooks we should execute
		for _, hook := range c.Hooks {
			if hook.When != nil {
				if when == Before && hook.When.Before != nil {
					if stage == StageDeployments && hook.When.Before.Deployments != "" && compareWhich(which, hook.When.Before.Deployments) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePurgeDeployments && hook.When.Before.PurgeDeployments != "" && compareWhich(which, hook.When.Before.PurgeDeployments) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.Before.Images != "" && compareWhich(which, hook.When.Before.Images) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageDependencies && hook.When.Before.Dependencies != "" && compareWhich(which, hook.When.Before.Dependencies) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePullSecrets && hook.When.Before.PullSecrets != "" && compareWhich(which, hook.When.Before.PullSecrets) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageInitialSync && hook.When.Before.InitialSync != "" && compareWhich(which, hook.When.Before.InitialSync) {
						hooksToExecute = append(hooksToExecute, hook)
					}
				} else if when == After && hook.When.After != nil {
					if stage == StageDeployments && hook.When.After.Deployments != "" && compareWhich(which, hook.When.After.Deployments) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePurgeDeployments && hook.When.After.PurgeDeployments != "" && compareWhich(which, hook.When.After.PurgeDeployments) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.After.Images != "" && compareWhich(which, hook.When.After.Images) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageDependencies && hook.When.After.Dependencies != "" && compareWhich(which, hook.When.After.Dependencies) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePullSecrets && hook.When.After.PullSecrets != "" && compareWhich(which, hook.When.After.PullSecrets) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageInitialSync && hook.When.After.InitialSync != "" && compareWhich(which, hook.When.After.InitialSync) {
						hooksToExecute = append(hooksToExecute, hook)
					}
				} else if when == OnError && hook.When.OnError != nil {
					if stage == StageDeployments && hook.When.OnError.Deployments != "" && compareWhich(which, hook.When.OnError.Deployments) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePurgeDeployments && hook.When.OnError.PurgeDeployments != "" && compareWhich(which, hook.When.OnError.PurgeDeployments) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.OnError.Images != "" && compareWhich(which, hook.When.OnError.Images) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageDependencies && hook.When.OnError.Dependencies != "" && compareWhich(which, hook.When.OnError.Dependencies) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePullSecrets && hook.When.OnError.PullSecrets != "" && compareWhich(which, hook.When.OnError.PullSecrets) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageInitialSync && hook.When.OnError.InitialSync != "" && compareWhich(which, hook.When.OnError.InitialSync) {
						hooksToExecute = append(hooksToExecute, hook)
					}
				}
			}
		}

		// Execute hooks
		for _, hookConfig := range hooksToExecute {
			if !command.ShouldExecuteOnOS(hookConfig.OperatingSystem) {
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
			if hookConfig.Where.Container != nil {
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
			err := executeHook(context, hookConfig, hookWriter, e.config, e.dependencies, log, hook)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func compareWhich(current, fromConfig string) bool {
	current = strings.TrimSpace(current)
	splitted := strings.Split(fromConfig, ",")
	for _, s := range splitted {
		if strings.TrimSpace(s) == current {
			return true
		}
	}

	return false
}

func executeHook(ctx Context, hookConfig *latest.HookConfig, hookWriter io.Writer, config config.Config, dependencies []types.Dependency, log logpkg.Logger, hook Hook) error {
	hookLog := log
	if hookConfig.Silent {
		hookLog = logpkg.Discard
	}

	if hookConfig.Background {
		log.Infof("Execute hook '%s' in background", ansi.Color(hookName(hookConfig), "white+b"))
		go func() {
			err := hook.Execute(ctx, hookConfig, config, dependencies, hookLog)
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

	log.Infof("Execute hook '%s'", ansi.Color(hookName(hookConfig), "white+b"))
	err := hook.Execute(ctx, hookConfig, config, dependencies, hookLog)
	if err != nil {
		if hookConfig.Silent {
			return errors.Wrapf(err, "in hook '%s': %s", ansi.Color(hookName(hookConfig), "white+b"), hookWriter.(*bytes.Buffer).String())
		}
		return errors.Wrapf(err, "in hook '%s'", ansi.Color(hookName(hookConfig), "white+b"))
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
	if hook.Upload != nil && hook.Where.Container != nil {
		localPath := "."
		if hook.Upload.LocalPath != "" {
			localPath = hook.Upload.LocalPath
		}
		containerPath := "."
		if hook.Upload.ContainerPath != "" {
			containerPath = hook.Upload.ContainerPath
		}

		if hook.Where.Container.Pod != "" {
			return fmt.Sprintf("copy %s to pod %s", localPath, hook.Where.Container.Pod)
		}
		if len(hook.Where.Container.LabelSelector) > 0 {
			return fmt.Sprintf("copy %s to selector %s", localPath, labels.Set(hook.Where.Container.LabelSelector).String())
		}
		if hook.Where.Container.ImageName != "" {
			return fmt.Sprintf("copy %s to imageName %s", localPath, hook.Where.Container.ImageName)
		}
		if hook.Where.Container.ImageSelector != "" {
			return fmt.Sprintf("copy %s to image %s", localPath, hook.Where.Container.ImageSelector)
		}

		return fmt.Sprintf("copy %s to %s", localPath, containerPath)
	}
	if hook.Download != nil && hook.Where.Container != nil {
		localPath := "."
		if hook.Download.LocalPath != "" {
			localPath = hook.Download.LocalPath
		}
		containerPath := "."
		if hook.Download.ContainerPath != "" {
			containerPath = hook.Download.ContainerPath
		}

		if hook.Where.Container.Pod != "" {
			return fmt.Sprintf("download from pod %s to %s", hook.Where.Container.Pod, localPath)
		}
		if len(hook.Where.Container.LabelSelector) > 0 {
			return fmt.Sprintf("download from selector %s to %s", labels.Set(hook.Where.Container.LabelSelector).String(), localPath)
		}
		if hook.Where.Container.ImageName != "" {
			return fmt.Sprintf("download from imageName %s to %s", hook.Where.Container.ImageName, localPath)
		}
		if hook.Where.Container.ImageSelector != "" {
			return fmt.Sprintf("download from image %s to %s", hook.Where.Container.ImageSelector, localPath)
		}

		return fmt.Sprintf("download from container:%s to local:%s", containerPath, localPath)
	}
	if hook.Logs != nil && hook.Where.Container != nil {
		if hook.Where.Container.Pod != "" {
			return fmt.Sprintf("logs from pod %s", hook.Where.Container.Pod)
		}
		if len(hook.Where.Container.LabelSelector) > 0 {
			return fmt.Sprintf("logs from selector %s", labels.Set(hook.Where.Container.LabelSelector).String())
		}
		if hook.Where.Container.ImageName != "" {
			return fmt.Sprintf("logs from imageName %s", hook.Where.Container.ImageName)
		}
		if hook.Where.Container.ImageSelector != "" {
			return fmt.Sprintf("logs from image %s", hook.Where.Container.ImageSelector)
		}

		return "logs from first container found"
	}
	if hook.Wait != nil && hook.Where.Container != nil {
		if hook.Where.Container.Pod != "" {
			return fmt.Sprintf("wait for pod %s", hook.Where.Container.Pod)
		}
		if len(hook.Where.Container.LabelSelector) > 0 {
			return fmt.Sprintf("wait for selector %s", labels.Set(hook.Where.Container.LabelSelector).String())
		}
		if hook.Where.Container.ImageName != "" {
			return fmt.Sprintf("wait for imageName %s", hook.Where.Container.ImageName)
		}
		if hook.Where.Container.ImageSelector != "" {
			return fmt.Sprintf("wait for image %s", hook.Where.Container.ImageSelector)
		}

		return "wait for everything"
	}
	return "hook"
}
