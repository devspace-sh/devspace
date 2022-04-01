package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/helper/server"
	"github.com/spf13/cobra"
)

// UpstreamCmd holds the upstream cmd flags
type UpstreamCmd struct {
	FileChangeCmd  string
	FileChangeArgs []string

	DirCreateCmd  string
	DirCreateArgs []string

	BatchCmd  string
	BatchArgs []string

	OverridePermissions bool
	Exclude             []string
}

// NewUpstreamCmd creates a new upstream command
func NewUpstreamCmd() *cobra.Command {
	cmd := &UpstreamCmd{}
	upstreamCmd := &cobra.Command{
		Use:   "upstream",
		Short: "Starts the upstream sync server",
		Args:  cobra.ExactArgs(1),
		RunE:  cmd.Run,
	}

	upstreamCmd.Flags().StringVar(&cmd.FileChangeCmd, "filechangecmd", "", "Command that should be run during a file create or update")
	upstreamCmd.Flags().StringSliceVar(&cmd.FileChangeArgs, "filechangeargs", []string{}, "Args that should be used for the command that is run during a file create or update")

	upstreamCmd.Flags().StringVar(&cmd.DirCreateCmd, "dircreatecmd", "", "Command that should be run during a directory create")
	upstreamCmd.Flags().StringSliceVar(&cmd.DirCreateArgs, "dircreateargs", []string{}, "Args that should be used for the command that is run during a directory create")

	upstreamCmd.Flags().BoolVar(&cmd.OverridePermissions, "override-permissions", false, "If enabled will override file permissions")
	upstreamCmd.Flags().StringSliceVar(&cmd.Exclude, "exclude", []string{}, "The exclude paths for upstream watching")
	return upstreamCmd
}

// Run runs the command logic
func (cmd *UpstreamCmd) Run(cobraCmd *cobra.Command, args []string) error {
	absolutePath, err := ensurePath(args)
	if err != nil {
		return err
	}

	return server.StartUpstreamServer(os.Stdin, os.Stdout, &server.UpstreamOptions{
		UploadPath:  absolutePath,
		ExludePaths: cmd.Exclude,

		FileChangeCmd:  cmd.FileChangeCmd,
		FileChangeArgs: cmd.FileChangeArgs,

		DirCreateCmd:  cmd.DirCreateCmd,
		DirCreateArgs: cmd.DirCreateArgs,

		OverridePermission: cmd.OverridePermissions,
		ExitOnClose:        true,
		Ping:               true,
	})
}

func ensurePath(args []string) (string, error) {
	// Create the directory if it does not exist
	path := args[0]
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return "", err
		}
	}

	// we have to resolve the real local path, because the watcher gives us the real path always
	realLocalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}

	absolutePath, err := filepath.Abs(realLocalPath)
	if err != nil {
		return "", err
	}

	if absolutePath == "/" && path != "/" {
		return "", fmt.Errorf("you are trying to sync the complete container root (/). By default this is not allowed, because this usually leads to unwanted behaviour. Please specify the correct container directory via the `--path` flag or `.path: localPath:/remotePath` option")
	}

	return absolutePath, nil
}
