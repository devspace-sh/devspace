package sync

import (
	"os"

	"github.com/loft-sh/devspace/helper/server"
	"github.com/spf13/cobra"
)

// DownstreamCmd holds the downstream cmd flags
type DownstreamCmd struct {
	Exclude []string

	Throttle int64

	Polling        bool
	RecursiveWatch bool
}

// NewDownstreamCmd creates a new downstream command
func NewDownstreamCmd() *cobra.Command {
	cmd := &DownstreamCmd{}
	downstreamCmd := &cobra.Command{
		Use:   "downstream",
		Short: "Starts the downstream sync server",
		Args:  cobra.ExactArgs(1),
		RunE:  cmd.Run,
	}

	downstreamCmd.Flags().StringSliceVar(&cmd.Exclude, "exclude", []string{}, "The exclude paths for downstream watching")
	downstreamCmd.Flags().Int64Var(&cmd.Throttle, "throttle", 5, "The amount of milliseconds to throttle change detection per 100 files")
	downstreamCmd.Flags().BoolVar(&cmd.Polling, "polling", false, "If true, DevSpace will use polling instead of inotify")
	downstreamCmd.Flags().BoolVar(&cmd.RecursiveWatch, "recursive-watch", true, "If false, DevSpace will not watch recursively")
	return downstreamCmd
}

// Run runs the command logic
func (cmd *DownstreamCmd) Run(cobraCmd *cobra.Command, args []string) error {
	absolutePath, err := ensurePath(args)
	if err != nil {
		return err
	}

	return server.StartDownstreamServer(os.Stdin, os.Stdout, &server.DownstreamOptions{
		RemotePath:   absolutePath,
		ExcludePaths: cmd.Exclude,

		Throttle:         cmd.Throttle,
		Polling:          cmd.Polling,
		ExitOnClose:      true,
		NoRecursiveWatch: !cmd.RecursiveWatch,
		Ping:             true,
	})
}
