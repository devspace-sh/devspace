package pipeline

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/pipelinehandler"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"io"
	"mvdan.cc/sh/v3/expand"
	"os"
	"sync"
)

type Job struct {
	Pipeline types.Pipeline
	Config   *latest.Pipeline

	m sync.Mutex
	t *tomb.Tomb
}

func (j *Job) Done() <-chan struct{} {
	j.m.Lock()
	defer j.m.Unlock()

	if j.t != nil {
		return j.t.Dead()
	}

	done := make(chan struct{})
	close(done)
	return done
}

func (j *Job) Terminated() bool {
	j.m.Lock()
	defer j.m.Unlock()

	if j.t != nil {
		return j.t.Terminated()
	}

	return false
}

func (j *Job) Stop() error {
	j.m.Lock()
	t := j.t
	j.m.Unlock()

	if t == nil {
		return nil
	}

	t.Kill(nil)
	return t.Wait()
}

func (j *Job) Run(ctx devspacecontext.Context, args []string, environ expand.Environ) error {
	if ctx.IsDone() {
		return ctx.Context().Err()
	}

	j.m.Lock()
	ctx = ctx.WithContext(j.t.Context(ctx.Context()))
	t := j.t
	j.m.Unlock()

	t.Go(func() error {
		// start the actual job
		done := t.NotifyGo(func() error {
			return j.execute(ctx, args, t, environ)
		})

		// wait until job is dying
		select {
		case <-ctx.Context().Done():
			return nil
		case <-done:
		}

		// check if errored
		if !t.Alive() {
			return t.Err()
		}

		return nil
	})

	return t.Wait()
}

func (j *Job) execute(ctx devspacecontext.Context, args []string, parent *tomb.Tomb, environ expand.Environ) error {
	ctx = ctx.WithLogger(ctx.Log())
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutWriter.Close()

	parent.Go(func() error {
		s := scanner.NewScanner(stdoutReader)
		for s.Scan() {
			ctx.Log().Info(s.Text())
		}
		return nil
	})

	stderrReader, stderrWriter := io.Pipe()
	defer stderrWriter.Close()

	parent.Go(func() error {
		s := scanner.NewScanner(stderrReader)
		for s.Scan() {
			ctx.Log().Warn(s.Text())
		}
		return nil
	})

	handler := pipelinehandler.NewPipelineExecHandler(ctx, stdoutWriter, stderrWriter, j.Pipeline)
	_, err := engine.ExecutePipelineShellCommand(ctx.Context(), j.Config.Run, args, ctx.WorkingDir(), j.Config.ContinueOnError, stdoutWriter, stderrWriter, os.Stdin, environ, handler)
	return err
}
