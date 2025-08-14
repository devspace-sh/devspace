package commands

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/types"
	"mvdan.cc/sh/v3/interp"
)

var errXArgsUsage = errors.New(`usage: xargs [utility [argument ...]]`)

type XArgsOptions struct {
	Delimiter string
}

var XArgsFocusCommands = map[string]bool{}

func XArgs(ctx context.Context, args []string, handler types.ExecHandler) error {
	options := &XArgsOptions{
		Delimiter: " ",
	}

	args, err := parseXArgsOptions(args, options)
	if err != nil {
		return err
	} else if len(args) == 0 {
		return errXArgsUsage
	} else if !XArgsFocusCommands[args[0]] {
		newArgs := []string{"xargs"}
		newArgs = append(newArgs, args...)
		return interp.DefaultExecHandler(2*time.Second)(ctx, newArgs)
	}

	hc := interp.HandlerCtx(ctx)
	out, err := io.ReadAll(hc.Stdin)
	if err != nil {
		return err
	}

	addArgs := strings.Split(string(out), "\n")
	for _, addArg := range addArgs {
		splitted := strings.Split(addArg, options.Delimiter)
		for _, a := range splitted {
			a = strings.TrimSpace(a)
			if a == "" {
				continue
			}

			args = append(args, a)
		}
	}
	return handler.ExecHandler(ctx, args)
}

func parseXArgsOptions(args []string, options *XArgsOptions) ([]string, error) {
	// check args for flags
	startAt := 0
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) > 0 && arg[0] == '-' {
			startAt++
			arg = arg[1:]

			switch arg {
			case "d", "-delimiter":
				if i+1 == len(args) {
					return nil, errXArgsUsage
				}

				i++
				startAt++
				options.Delimiter = args[i]
			default:
				return nil, errXArgsUsage
			}

			continue
		}

		break
	}

	return args[startAt:], nil
}
