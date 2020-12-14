package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"io"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/command"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	dockerterm "github.com/docker/docker/pkg/term"
)

const (
	KubeContextEnv   = "DEVSPACE_HOOK_KUBE_CONTEXT"
	KubeNamespaceEnv = "DEVSPACE_HOOK_KUBE_NAMESPACE"
	ErrorEnv         = "DEVSPACE_HOOK_ERROR"
	OsArgsEnv        = "DEVSPACE_HOOK_OS_ARGS"
)

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
		for _, hook := range hooksToExecute {
			if command.ShouldExecuteOnOS(hook.OperatingSystem) == false {
				continue
			}

			// Determine output writer
			var writer io.Writer
			if log == logpkg.GetInstance() {
				writer = stdout
			} else {
				writer = log
			}

			// Where to execute
			execute := executeLocally
			if hook.Where.Container != nil {
				execute = executeInContainer
			}

			// Execute the hook
			err := executeHook(context, hook, writer, log, execute)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func executeHook(ctx Context, hook *latest.HookConfig, writer io.Writer, log logpkg.Logger, execute func(Context, *latest.HookConfig, io.Writer, logpkg.Logger) error) error {
	var (
		hookLog    logpkg.Logger
		hookWriter io.Writer
	)
	if hook.Silent {
		hookLog = logpkg.Discard
		hookWriter = &bytes.Buffer{}
	} else {
		hookLog = log
		hookWriter = writer
	}

	if hook.Background {
		log.Infof("Execute hook '%s' in background", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"))
		go func() {
			err := execute(ctx, hook, hookWriter, hookLog)
			if err != nil {
				if hook.Silent {
					log.Warnf("Error executing hook '%s' in background: %s %v", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"), hookWriter.(*bytes.Buffer).String(), err)
				} else {
					log.Warnf("Error executing hook '%s' in background: %v", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"), err)
				}
			}
		}()

		return nil
	}

	log.Infof("Execute hook '%s'", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"))
	err := execute(ctx, hook, hookWriter, hookLog)
	if err != nil {
		if hook.Silent {
			return errors.Wrapf(err, "in hook '%s': %s", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"), hookWriter.(*bytes.Buffer).String())
		} else {
			return errors.Wrapf(err, "in hook '%s'", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"))
		}
	}

	return nil
}

func executeLocally(ctx Context, hook *latest.HookConfig, writer io.Writer, log logpkg.Logger) error {
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

	err = command.ExecuteCommandWithEnv(hook.Command, hook.Args, writer, writer, extraEnv)
	if err != nil {
		return err
	}

	return nil
}

func executeInContainer(ctx Context, hook *latest.HookConfig, writer io.Writer, log logpkg.Logger) error {
	if ctx.Client == nil {
		return errors.Errorf("Cannot execute hook '%s': kube client is not initialized", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"))
	}

	var imageSelector []string
	if hook.Where.Container.ImageName != "" {
		if ctx.Config == nil || ctx.Cache == nil {
			return errors.Errorf("Cannot execute hook '%s': config is not loaded", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"))
		}

		imageSelector = targetselector.ImageSelectorFromConfig(hook.Where.Container.ImageName, ctx.Config, ctx.Cache)
	}

	if hook.Where.Container.Wait == nil || *hook.Where.Container.Wait == true {
		log.Infof("Waiting for running containers for hook '%s'", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"))

		timeout := time.Second * 120
		if hook.Where.Container.Timeout > 0 {
			timeout = time.Duration(hook.Where.Container.Timeout) * time.Second
		}

		err := wait.Poll(time.Second, timeout, func() (done bool, err error) {
			return executeInFoundContainer(ctx, hook, imageSelector, writer, log)
		})
		if err != nil {
			if err == wait.ErrWaitTimeout {
				return errors.Errorf("timeout: couldn't find a running container")
			}

			return err
		}

		return nil
	}

	executed, err := executeInFoundContainer(ctx, hook, imageSelector, writer, log)
	if err != nil {
		return err
	} else if executed == false {
		log.Infof("Skip hook '%s', because no running containers were found", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"))
	}
	return nil
}

func executeInFoundContainer(ctx Context, hook *latest.HookConfig, imageSelector []string, writer io.Writer, log logpkg.Logger) (bool, error) {
	labelSelector := ""
	if len(hook.Where.Container.LabelSelector) > 0 {
		labelSelector = labels.Set(hook.Where.Container.LabelSelector).String()
	}

	podContainers, err := kubectl.NewFilterWithSort(ctx.Client, kubectl.SortPodsByNewest, kubectl.SortContainersByNewest).SelectContainers(context.TODO(), kubectl.Selector{
		ImageSelector:   imageSelector,
		LabelSelector:   labelSelector,
		Pod:             hook.Where.Container.Pod,
		ContainerName:   hook.Where.Container.ContainerName,
		Namespace:       hook.Where.Container.Namespace,
	})
	if err != nil {
		return false, err
	} else if len(podContainers) == 0 {
		return false, nil
	}
	
	// if any podContainer is not running we wait
	for _, podContainer := range podContainers {
		if targetselector.IsContainerRunning(podContainer) == false {
			return false, nil
		}
	}

	// execute the hook in the containers
	for _, podContainer := range podContainers {
		cmd := []string{hook.Command}
		cmd = append(cmd, hook.Args...)

		log.Infof("Execute hook '%s' in container '%s/%s/%s'", ansi.Color(fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " ")), "white+b"), podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name)
		err = ctx.Client.ExecStream(&kubectl.ExecStreamOptions{
			Pod:       podContainer.Pod,
			Container: podContainer.Container.Name,
			Command:   cmd,
			Stdout:    writer,
			Stderr:    writer,
		})
		if err != nil {
			return false, errors.Errorf("error in container '%s/%s/%s': %v", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
		}
	}

	return true, nil
}
