package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar"
	"github.com/jessevdk/go-flags"
	types2 "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/loft-sh/notify"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/interp"
)

type RunWatchOptions struct {
	FailOnError bool `long:"fail-on-error" description:"If true the command will fail on an error while running the sub command"`

	SkipInitial   bool     `long:"skip-initial" description:"If true will not execute the command immediately."`
	Silent        bool     `long:"silent" description:"If true will not print any warning about restarting the command."`
	SkipAndSilent bool     `long:"skip-and-silent" short:"s" description:"If enabled will not print when the command is restarted and not execute the command initially."`
	Exclude       []string `long:"exclude" short:"e" description:"The paths to ignore. Can be patterns in the form of ./**/my-file.txt"`
	Paths         []string `long:"path" short:"p" description:"The paths to watch. Can be patterns in the form of ./**/my-file.txt"`
}

func RunWatch(ctx context.Context, args []string, handler types2.ExecHandler, log log.Logger) error {
	command := []string{}

	// Separately handle the `--` catchall flag
	// in order to have clean arguments for the parsing below
	commandINdex := indexOf("--", args)
	if commandINdex > 0 {
		command = args[commandINdex+1:]
	}

	options := &RunWatchOptions{}
	args, err := flags.ParseArgs(options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}
	if len(options.Paths) == 0 {
		return fmt.Errorf("usage: run_watch --path MY_PATH -- my_command")
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: run_watch --path MY_PATH -- my_command")
	}
	if len(args) > len(command) {
		// if we have more args left thant the one after "--" then we have some invalid flags inside
		return fmt.Errorf("invalid flags: %v, usage: run_watch --path MY_PATH --path 'MY/**/GLOB*/PATH' -- my_command", command)
	}

	w := &watcher{
		options: *options,
	}

	return w.Watch(ctx, func(ctx context.Context) error {
		return handler.ExecHandler(ctx, args)
	}, log)
}

type watcher struct {
	options RunWatchOptions
}

func (w *watcher) Watch(ctx context.Context, action func(ctx context.Context) error, log log.Logger) error {
	if w.options.SkipAndSilent {
		w.options.Silent = true
		w.options.SkipInitial = true
	}

	// prepare patterns
	patterns := w.options.Paths
	for i, p := range patterns {
		patterns[i] = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(p), "./"), "/")
	}

	// prepare excludes
	excludes := w.options.Exclude
	for i, e := range excludes {
		excludes[i] = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(e), "./"), "/")
	}

	// get folders from patterns
	pathsToWatch := map[string]bool{}
	for i, p := range patterns {
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
		} else {
			patterns[i] = strings.TrimPrefix(patterns[i], targetPath+"/")
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
	return w.handleCommand(ctx, patterns, excludes, action, globalChannel)
}

func (w *watcher) handleCommand(ctx context.Context, patterns []string, excludes []string, action func(ctx context.Context) error, events chan string) error {
	hc := interp.HandlerCtx(ctx)
	var t *tomb.Tomb
	if !w.options.SkipInitial {
		t = w.startCommand(ctx, action)
	} else {
		t = &tomb.Tomb{}
		t.Go(func() error { return nil })
	}
	numEvents := 0
	lastChange := ""
	for {
		select {
		case <-ctx.Done():
			t.Kill(nil)
			<-t.Dead()
			return nil
		case e := <-events:
			// check if match
			hasMatched := false
			for _, p := range patterns {
				hasMatched, _ = doublestar.Match(p, e)
				if hasMatched {
					break
				}
			}

			if hasMatched {
				excluded := false
				for _, excludePath := range excludes {
					excluded, _ = doublestar.Match(excludePath, e)
					if excluded {
						break
					}
				}

				if !excluded {
					numEvents++
					lastChange = e
				}
			}
		case <-time.After(time.Millisecond * 1200):
			if numEvents > 0 {
				// kill application and wait for exit
				if !w.options.Silent {
					_, _ = hc.Stderr.Write([]byte(fmt.Sprintf("\n%s Restarting command because '%s' has changed...\n\n", ansi.Color("warn", "red+b"), lastChange)))
				}
				t.Kill(nil)
				select {
				case <-ctx.Done():
					return nil
				case <-t.Dead():
				}

				// restart the command
				t = w.startCommand(ctx, action)
				numEvents = 0
				lastChange = ""
			}
		}

		// check if terminated
		if w.options.FailOnError && t.Terminated() && t.Err() != nil {
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

func indexOf(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1
}
