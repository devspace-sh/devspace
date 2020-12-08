package hook

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/mgutz/ansi"
	"io"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/command"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	dockerterm "github.com/docker/docker/pkg/term"
)

const (
	KubeContextEnv   = "DEVSPACE_HOOK_KUBE_CONTEXT"
	KubeNamespaceEnv = "DEVSPACE_HOOK_KUBE_NAMESPACE"
	ErrorEnv         = "DEVSPACE_HOOK_ERROR"
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
type When int

const (
	// Before is used to tell devspace to execute a hook before a certain stage
	Before When = iota
	// After is used to tell devspace to execute a hook after a certain stage
	After
	// OnError is used to tell devspace to execute a hook after a certain error occured
	OnError
)

// Stage is the type that defines the stage at when to execute a hook
type Stage int

const (
	// StageImages is the image building stage
	StageImages Stage = iota
	// StageDeployments is the deploying stage
	StageDeployments
	// StageDependencies is the dependency stage
	StageDependencies
	// StagePullSecrets is the pull secrets stage
	StagePullSecrets
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

		// Create extra env variables
		extraEnv := map[string]string{}
		if context.Client != nil {
			extraEnv[KubeContextEnv] = context.Client.CurrentContext()
			extraEnv[KubeNamespaceEnv] = context.Client.Namespace()
		}
		if when == OnError && context.Error != nil {
			extraEnv[ErrorEnv] = context.Error.Error()
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

			log.Infof("Execute hook: %s", ansi.Color(fmt.Sprintf("%s '%s'", hook.Command, strings.Join(hook.Args, "' '")), "white+b"))
			err := command.ExecuteCommandWithEnv(hook.Command, hook.Args, writer, writer, extraEnv)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
