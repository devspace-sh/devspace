package pipeline

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/registry"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/env"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/pkg/errors"
	"io"
	"mvdan.cc/sh/v3/interp"
	"os"
	"strings"
	"sync"
)

type Job struct {
	DependencyRegistry registry.DependencyRegistry
	DevPodManager      devpod.Manager

	Config *latest.Pipeline

	m       sync.Mutex
	started bool
	t       *tomb.Tomb
}

func (j *Job) Run(ctx *devspacecontext.Context) error {
	if ctx.IsDone() {
		return ctx.Context.Err()
	}

	j.m.Lock()
	defer j.m.Unlock()

	if j.started {
		return j.t.Err()
	}

	j.started = true

	tombCtx := j.t.Context(ctx.Context)
	ctx = ctx.WithContext(tombCtx)
	j.t.Go(func() error {
		// start the actual job
		done := j.t.NotifyGo(func() error {
			return j.doWork(ctx)
		})

		// wait until job is dying
		select {
		case <-ctx.Context.Done():
			return nil
		case <-done:
		}

		// check if errored
		if !j.t.Alive() {
			return j.t.Err()
		}

		// if rerun we should watch here
		if j.Config.Rerun != nil {
			// TODO: watch and restart job here
			return nil
		}

		return nil
	})

	return j.t.Wait()
}

func (j *Job) doWork(ctx *devspacecontext.Context) error {
	// loop over steps and execute them
	for i, step := range j.Config.Steps {
		var (
			execute = true
			err     error
		)
		if step.If != "" {
			execute, err = j.shouldExecuteStep(ctx, &step)
			if err != nil {
				return errors.Wrapf(err, "error checking if at step %d", i)
			}
		}
		if execute {
			err = j.executeStep(ctx, &step)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (j *Job) shouldExecuteStep(ctx *devspacecontext.Context, step *latest.PipelineStep) (bool, error) {
	// check if step should be rerun
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	handler := engine.NewExecHandler(ctx, nil, j.DependencyRegistry, j.DevPodManager, false)
	_, err := engine.ExecuteShellCommand(ctx.Context, step.Run, os.Args[1:], step.Directory, false, stdout, stderr, env.NewVariableEnvProvider(ctx.Config, ctx.Dependencies, step.Env), handler)
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

func (j *Job) executeStep(ctx *devspacecontext.Context, step *latest.PipelineStep) error {
	ctx = ctx.WithLogger(ctx.Log.WithoutPrefix())
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutWriter.Close()
	go func() {
		s := scanner.NewScanner(stdoutReader)
		for s.Scan() {
			ctx.Log.Info(s.Text())
		}
	}()

	handler := engine.NewExecHandler(ctx, stdoutWriter, j.DependencyRegistry, j.DevPodManager, true)
	_, err := engine.ExecuteShellCommand(ctx.Context, step.Run, os.Args[1:], step.Directory, step.ContinueOnError, stdoutWriter, stdoutWriter, env.NewVariableEnvProvider(ctx.Config, ctx.Dependencies, step.Env), handler)
	return err
}
