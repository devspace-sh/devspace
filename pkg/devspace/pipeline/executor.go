package pipeline

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/pkg/errors"
	"io"
	"mvdan.cc/sh/v3/interp"
	"os"
	"strings"
)

type Executor interface {
	ExecutePipeline(pipeline *latest.Pipeline, log log.Logger) error
}

func NewExecutor(conf config.Config, dependencies []types.Dependency, client kubectl.Client) Executor {
	return &executor{
		devSpaceConfig: conf,
		dependencies:   dependencies,
		client:         client,
	}
}

type executor struct {
	devSpaceConfig config.Config
	dependencies   []types.Dependency
	client         kubectl.Client
}

func (e *executor) ExecutePipeline(pipeline *latest.Pipeline, logger log.Logger) error {
	for i, step := range pipeline.Steps {
		name := fmt.Sprintf("[step:%d] ", i)
		if step.Name != "" {
			name = "[" + step.Name + "] "
		}

		prefixLogger := log.NewDefaultPrefixLogger(name, logger)
		if step.If != "" {
			shouldExecute, err := e.shouldExecuteStep(&step, prefixLogger)
			if err != nil {
				return errors.Wrapf(err, "execute if for step %s", name)
			} else if !shouldExecute {
				continue
			}
		}

		err := e.executeStep(&step, prefixLogger)
		if err != nil {
			return errors.Wrapf(err, "execute step %s", name)
		}
	}

	return nil
}

func (e *executor) shouldExecuteStep(step *latest.PipelineStep, logger log.Logger) (bool, error) {
	// check if step should be rerun
	handler := engine.NewExecHandler(e.devSpaceConfig, e.dependencies, e.client, logger)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	_, err := engine.ExecuteShellCommand(step.Command, os.Args[1:], step.Directory, stdout, stderr, step.Env, handler)
	if err != nil {
		if status, ok := interp.IsExitStatus(err); ok && status == 1 {
			return false, nil
		}

		return false, fmt.Errorf("error: %v (stdout: %s, stderr: %s)", err, stdout.String(), stderr.String())
	} else if strings.TrimSpace(stdout.String()) == "false" {
		return false, nil
	}

	return true, nil
}

func (e *executor) executeStep(step *latest.PipelineStep, logger log.Logger) error {
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutWriter.Close()
	go func() {
		s := scanner.NewScanner(stdoutReader)
		for s.Scan() {
			logger.Info(s.Text())
		}
	}()

	handler := engine.NewExecHandler(e.devSpaceConfig, e.dependencies, e.client, logger)
	_, err := engine.ExecuteShellCommand(step.Command, os.Args[1:], step.Directory, stdoutWriter, stdoutWriter, step.Env, handler)
	return err
}
