package commands

import (
	"context"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/jessevdk/go-flags"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/loft-sh/notify"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/interp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WatchOptions struct {
	FailOnError bool `long:"fail-on-error" description:"If true the command will fail on an error while running the sub command"`

	Paths []string `long:"path" short:"p" description:"The paths to watch. Can be patterns in the form of ./**/my-file.txt"`
}

func Watch(devCtx *devspacecontext.Context, pipeline types.Pipeline, args []string, newHandler NewHandlerFn) error {
	devCtx.Log.Debugf("watch %s", strings.Join(args, " "))
	options := &WatchOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}
	if len(options.Paths) == 0 {
		return fmt.Errorf("usage: watch --path MY_PATH -- my_command")
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: watch --path MY_PATH -- my_command")
	}

	w := &watcher{}
	hc := interp.HandlerCtx(devCtx.Context)
	return w.Watch(devCtx.Context, options.Paths, options.FailOnError, func(ctx context.Context) error {
		devCtx := devCtx.WithContext(ctx)
		_, err := engine.ExecutePipelineShellCommand(ctx, args[0]+" $@", args[1:], hc.Dir, false, hc.Stdout, hc.Stderr, hc.Stdin, hc.Env, newHandler(devCtx, hc.Stdout, pipeline))
		return err
	}, devCtx.Log)
}

type watcher struct{}

func (w *watcher) Watch(ctx context.Context, patterns []string, failOnError bool, action func(ctx context.Context) error, log log.Logger) error {
	// prepare patterns
	for i, p := range patterns {
		patterns[i] = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(p), "./"), "/")
	}

	// get folders from patterns
	pathsToWatch := map[string]bool{}
	for _, p := range patterns {
		patternsSplitted := strings.Split(filepath.ToSlash(p), "/")
		lastIndex := len(patternsSplitted) - 1
		for i, s := range patternsSplitted {
			if strings.Contains(s, "*") {
				lastIndex = i
				break
			}
		}

		targetPath := strings.Join(patternsSplitted[:lastIndex], "/")
		if targetPath == "" {
			targetPath = "."
		}

		absolutePath, err := filepath.Abs(filepath.FromSlash(targetPath))
		if err != nil {
			return errors.Wrap(err, "error resolving "+targetPath)
		}

		absolutePath, err = filepath.EvalSymlinks(absolutePath)
		if err != nil {
			return errors.Wrap(err, "eval symlinks")
		}

		pathsToWatch[absolutePath] = true
	}

	watchTree := notify.NewTree()
	defer watchTree.Close()

	globalChannel := make(chan string, 100)
	for targetPath := range pathsToWatch {
		stat, err := os.Stat(targetPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("cannot watch %s as the directory or file must exist", targetPath)
			}

			return errors.Wrap(err, "stat watch path "+targetPath)
		}

		// watch recursive if target path is a directory
		watchPath := targetPath
		if stat.IsDir() {
			watchPath = filepath.Join(watchPath, "...")
		}

		// start watching
		eventChannel := make(chan notify.EventInfo, 100)
		log.Debugf("Start watching %v", targetPath)
		err = watchTree.Watch(watchPath, eventChannel, func(s string) bool {
			return false
		}, notify.All)
		if err != nil {
			return errors.Wrap(err, "start watching "+targetPath)
		}
		defer watchTree.Stop(eventChannel)

		go func(base string, eventChannel chan notify.EventInfo) {
			for {
				select {
				case <-ctx.Done():
					return
				case e := <-eventChannel:
					// make relative
					relPath, err := filepath.Rel(base, e.Path())
					if err != nil {
						log.Debugf("error converting path %s: %v", e.Path(), err)
					} else {
						globalChannel <- filepath.ToSlash(relPath)
					}
				}
			}
		}(targetPath, eventChannel)
	}

	// start command
	return w.handleCommand(ctx, patterns, failOnError, action, globalChannel, log)
}

func (w *watcher) handleCommand(ctx context.Context, patterns []string, failOnError bool, action func(ctx context.Context) error, events chan string, log log.Logger) error {
	t := w.startCommand(ctx, action)
	numEvents := 0
	for {
		select {
		case <-ctx.Done():
			t.Kill(nil)
			<-t.Dead()
			return nil
		case e := <-events:
			// check if match
			for _, p := range patterns {
				hasMatched, _ := doublestar.Match(p, e)
				if hasMatched {
					numEvents++
					break
				}
			}
		case <-time.After(time.Second * 2):
			if numEvents > 0 {
				// kill application and wait for exit
				log.Infof("Restarting command...")
				t.Kill(nil)
				select {
				case <-ctx.Done():
					return nil
				case <-t.Dead():
				}

				// restart the command
				t = w.startCommand(ctx, action)
				numEvents = 0
			}
		}

		// check if terminated
		if failOnError && t.Terminated() && t.Err() != nil {
			return t.Err()
		}
	}
}

func (w *watcher) startCommand(ctx context.Context, action func(ctx context.Context) error) *tomb.Tomb {
	t, tombCtx := tomb.WithContext(ctx)
	t.Go(func() error {
		return action(tombCtx)
	})
	return t
}
