package hook

import (
	"fmt"
	"io"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/command"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"

	dockerterm "github.com/docker/docker/pkg/term"
)

// Executer executes configured commands locally
type Executer interface {
	Execute(when When, stage Stage, which string) error
}

type executer struct {
	config *latest.Config
	log    logpkg.Logger
}

// NewExecuter creates an instance of Executer for the specified config
func NewExecuter(config *latest.Config, log logpkg.Logger) Executer {
	return &executer{
		config: config,
		log:    log,
	}
}

// When is the type that is used to tell devspace when relatively to a stage a hook should be executed
type When int

const (
	// Before is used to tell devspace to execute a hook before a certain stage
	Before When = iota
	// After is used to tell devspace to execute a hook after a certain stage
	After
)

// Stage is the type that defines the stage at when to execute a hook
type Stage int

const (
	// StageImages is the image building stage
	StageImages Stage = iota
	// StageDeployments is the deploying stage
	StageDeployments
)

// All is used to tell devspace to execute a hook before or after all images, deployments
const All = "all"

var (
	_, stdout, stderr = dockerterm.StdStreams()
)

// Execute executes hooks at a specific time
func (e *executer) Execute(when When, stage Stage, which string) error {
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
					}
				} else if when == After && hook.When.After != nil {
					if stage == StageDeployments && hook.When.After.Deployments != "" && strings.TrimSpace(hook.When.After.Deployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.After.Images != "" && strings.TrimSpace(hook.When.After.Images) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					}
				}
			}
		}

		// Execute hooks
		for _, hook := range hooksToExecute {
			// Build arguments
			args := []string{}
			if hook.Args != nil {
				for _, arg := range hook.Args {
					args = append(args, arg)
				}
			}

			cmd := command.NewStreamCommand(hook.Command, args)

			// Determine output writer
			var writer io.Writer
			if e.log == logpkg.GetInstance() {
				writer = stdout
			} else {
				writer = e.log
			}

			e.log.Infof("Execute hook: %s", ansi.Color(fmt.Sprintf("%s '%s'", hook.Command, strings.Join(args, "' '")), "white+b"))
			err := cmd.Run(writer, writer, nil)
			if err != nil {
				return errors.Errorf("Error executing hook: %v", err)
			}
		}
	}

	return nil
}
