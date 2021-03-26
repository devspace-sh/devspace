package hook

import (
	"bytes"
	"fmt"
	dockerterm "github.com/docker/docker/pkg/term"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/command"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"io"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
	"time"
)

const (
	KubeContextEnv   = "DEVSPACE_HOOK_KUBE_CONTEXT"
	KubeNamespaceEnv = "DEVSPACE_HOOK_KUBE_NAMESPACE"
	ErrorEnv         = "DEVSPACE_HOOK_ERROR"
	OsArgsEnv        = "DEVSPACE_HOOK_OS_ARGS"
)

// Hook is an interface to execute a specific hook type
type Hook interface {
	Execute(ctx Context, hook *latest.HookConfig, log logpkg.Logger) error
}

// Executer executes configured commands locally
type Executer interface {
	OnError(stage Stage, whichs []string, context Context, log logpkg.Logger)
	Execute(when When, stage Stage, which string, context Context, log logpkg.Logger) error
	ExecuteMultiple(when When, stage Stage, whichs []string, context Context, log logpkg.Logger) error
}

type executer struct {
	config *latest.Config
}

// NewExecuter creates an instance of Executer for the specified config
func NewExecuter(config *latest.Config) Executer {
	return &executer{
		config: config,
	}
}

// When is the type that is used to tell devspace when relatively to a stage a hook should be executed
type When string

const (
	// Before is used to tell devspace to execute a hook before a certain stage
	Before When = "before"
	// After is used to tell devspace to execute a hook after a certain stage
	After When = "after"
	// OnError is used to tell devspace to execute a hook after a certain error occured
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
)

// All is used to tell devspace to execute a hook before or after all images, deployments
const All = "all"

var (
	_, stdout, stderr = dockerterm.StdStreams()
)

// Context holds hook context information
type Context struct {
	Error  error
	Client kubectl.Client
	Config *latest.Config
	Cache  *generated.CacheConfig
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

// OnError is a convience method to handle the resulting error of a hook execution. Since we mostly return anyways after
// an error has occured this only prints additonal information why the hook failed
func (e *executer) OnError(stage Stage, whichs []string, context Context, log logpkg.Logger) {
	err := e.ExecuteMultiple(OnError, stage, whichs, context, log)
	if err != nil {
		log.Warnf("Hook failed: %v", err)
	}
}

// Execute executes hooks at a specific time
func (e *executer) Execute(when When, stage Stage, which string, context Context, log logpkg.Logger) error {
	if e.config.Hooks != nil && len(e.config.Hooks) > 0 {
		hooksToExecute := []*latest.HookConfig{}

		// Gather all hooks we should execute
		for _, hook := range e.config.Hooks {
			if hook.When != nil {
				if when == Before && hook.When.Before != nil {
					if stage == StageDeployments && hook.When.Before.Deployments != "" && strings.TrimSpace(hook.When.Before.Deployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePurgeDeployments && hook.When.Before.PurgeDeployments != "" && strings.TrimSpace(hook.When.Before.PurgeDeployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.Before.Images != "" && strings.TrimSpace(hook.When.Before.Images) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageDependencies && hook.When.Before.Dependencies != "" && strings.TrimSpace(hook.When.Before.Dependencies) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePullSecrets && hook.When.Before.PullSecrets != "" && strings.TrimSpace(hook.When.Before.PullSecrets) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					}
				} else if when == After && hook.When.After != nil {
					if stage == StageDeployments && hook.When.After.Deployments != "" && strings.TrimSpace(hook.When.After.Deployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePurgeDeployments && hook.When.After.PurgeDeployments != "" && strings.TrimSpace(hook.When.After.PurgeDeployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.After.Images != "" && strings.TrimSpace(hook.When.After.Images) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageDependencies && hook.When.After.Dependencies != "" && strings.TrimSpace(hook.When.After.Dependencies) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePullSecrets && hook.When.After.PullSecrets != "" && strings.TrimSpace(hook.When.After.PullSecrets) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					}
				} else if when == OnError && hook.When.OnError != nil {
					if stage == StageDeployments && hook.When.OnError.Deployments != "" && strings.TrimSpace(hook.When.OnError.Deployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePurgeDeployments && hook.When.OnError.PurgeDeployments != "" && strings.TrimSpace(hook.When.Before.PurgeDeployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.OnError.Images != "" && strings.TrimSpace(hook.When.OnError.Images) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageDependencies && hook.When.OnError.Dependencies != "" && strings.TrimSpace(hook.When.OnError.Dependencies) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePullSecrets && hook.When.OnError.PullSecrets != "" && strings.TrimSpace(hook.When.OnError.PullSecrets) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					}
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
			err := executeHook(context, hookConfig, hookWriter, log, hook)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func executeHook(ctx Context, hookConfig *latest.HookConfig, hookWriter io.Writer, log logpkg.Logger, hook Hook) error {
	hookLog := log
	if hookConfig.Silent {
		hookLog = logpkg.Discard
	}

	if hookConfig.Background {
		log.Infof("Execute hook '%s' in background", ansi.Color(hookName(hookConfig), "white+b"))
		go func() {
			err := hook.Execute(ctx, hookConfig, hookLog)
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
	err := hook.Execute(ctx, hookConfig, hookLog)
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
		return fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " "))
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

		return fmt.Sprintf("logs from first container found")
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

		return fmt.Sprintf("wait for everything")
	}
	return "hook"
}
