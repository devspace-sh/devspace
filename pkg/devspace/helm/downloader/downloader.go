package downloader

import (
	"context"
	"os"
	"strings"

	"github.com/loft-sh/utils/pkg/command"
	"github.com/loft-sh/utils/pkg/downloader/commands"
	"mvdan.cc/sh/v3/expand"
)

type helmCommand struct {
	commands.Command
}

func NewHelmCommand() commands.Command {
	return &helmCommand{
		Command: commands.NewHelmV3Command(),
	}
}

func (h *helmCommand) IsValid(ctx context.Context, path string) (bool, error) {
	out, err := command.Output(ctx, "", expand.ListEnviron(os.Environ()...), path, "version")
	if err != nil {
		return false, nil
	}

	outStr := string(out)
	return strings.Contains(outStr, `:"v3.`) || strings.Contains(outStr, `:"v4.`), nil
}
