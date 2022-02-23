package pipeline

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/pkg/errors"
	"io"
	"mvdan.cc/sh/v3/interp"
	"os"
	"strings"
	"sync"
)

type PipelineJob struct {
	Name          string
	DevPodManager devpod.Manager

	JobConfig *latest.PipelineJob
	Job       Job

	Parents  []*PipelineJob
	Children []*PipelineJob

	startOnce sync.Once
	err       error
}

func (j *PipelineJob) Run(ctx *devspacecontext.Context) error {
	j.startOnce.Do(func() {
		for _, parent := range j.Parents {
			select {
			case <-ctx.Context.Done():
				return
			case <-parent.Job.Done():
			}
		}

		// start the actual job
		err := j.Job.Start(ctx, j.doWork)
		if err != nil {
			j.err = err
			return
		}

		// wait until job is finished
		<-j.Job.Done()

		// check if error
		if j.Job.Error() != nil {
			j.err = j.Job.Error()
			return
		}

		// if rerun we should watch here
		if j.JobConfig.Rerun != nil {
			// TODO: watch and restart job here
			return
		}
	})
	return j.err
}

func (j *PipelineJob) doWork(ctx *devspacecontext.Context) error {
	// loop over steps and execute them
	for i, step := range j.JobConfig.Steps {
		execute, err := j.shouldExecuteStep(ctx, &step)
		if err != nil {
			return errors.Wrapf(err, "error checking if at step %d", i)
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

func (j *PipelineJob) shouldExecuteStep(ctx *devspacecontext.Context, step *latest.PipelineStep) (bool, error) {
	// check if step should be rerun
	handler := engine.NewExecHandler(ctx, j.DevPodManager)
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

func (j *PipelineJob) executeStep(ctx *devspacecontext.Context, step *latest.PipelineStep) error {
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutWriter.Close()
	go func() {
		s := scanner.NewScanner(stdoutReader)
		for s.Scan() {
			ctx.Log.Info(s.Text())
		}
	}()

	handler := engine.NewExecHandler(ctx, j.DevPodManager)
	_, err := engine.ExecuteShellCommand(step.Command, os.Args[1:], step.Directory, stdoutWriter, stdoutWriter, step.Env, handler)
	return err
}
