package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/sync/server"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: sync [--upstream|--downstream] [--version] [--exclude] [--filechangecmd] [--dircreatecmd] [--batchcmd] PATH\n")
	os.Exit(1)
}

var version string

func main() {
	var (
		excludePaths arrayFlags

		isUpstream   = flag.Bool("upstream", false, "If upstream should be started")
		isDownstream = flag.Bool("downstream", false, "If downstream should be started")
		showVersion  = flag.Bool("version", false, "Shows the version")

		fileChangeCmd  = flag.String("filechangecmd", "", "Command that should be run during a file create or update")
		fileChangeArgs arrayFlags

		dirCreateCmd  = flag.String("dircreatecmd", "", "Command that should be run during a directory create")
		dirCreateArgs arrayFlags

		batchCmd  = flag.String("batchcmd", "", "Command that should be run during a directory create")
		batchArgs arrayFlags
	)

	flag.Var(&excludePaths, "exclude", "The exclude paths for downstream watching")
	flag.Var(&fileChangeArgs, "filechangeargs", "Args that should be used for the command that is run during a file create or update")
	flag.Var(&dirCreateArgs, "dircreateargs", "Args that should be used for the command that is run during a directory create")
	flag.Var(&batchArgs, "batchargs", "Args that should be used for the command that is run after a batch of changes is processed")
	flag.Parse()

	// Should we just print the version?
	if *showVersion {
		if version == "" {
			version = "latest"
		}

		fmt.Printf("%s", version)
		os.Exit(0)
	}

	// parse the flags & arguments
	args := flag.Args()
	if len(args) != 1 {
		printUsage()
	}

	// Create the directory if it does not exist
	path := args[0]
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(1)
		}
	}

	// we have to resolve the real local path, because the watcher gives us the real path always
	realLocalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	absolutePath, err := filepath.Abs(realLocalPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	if absolutePath == "/" && path != "/" {
		fmt.Fprintf(os.Stderr, "You are trying to sync the complete container root (/). By default this is not allowed, because this usually leads to unwanted behaviour. Please specify the correct container directory via the `--container-path` flag or `.containerPath` option")
		os.Exit(1)
	}

	if *isDownstream {
		err := server.StartDownstreamServer(os.Stdin, os.Stdout, &server.DownstreamOptions{
			RemotePath:   absolutePath,
			ExcludePaths: excludePaths,

			ExitOnClose: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(1)
		}
	} else if *isUpstream {
		err := server.StartUpstreamServer(os.Stdin, os.Stdout, &server.UpstreamOptions{
			UploadPath:  absolutePath,
			ExludePaths: excludePaths,

			BatchCmd:  *batchCmd,
			BatchArgs: batchArgs,

			FileChangeCmd:  *fileChangeCmd,
			FileChangeArgs: fileChangeArgs,

			DirCreateCmd:  *dirCreateCmd,
			DirCreateArgs: dirCreateArgs,

			ExitOnClose: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(1)
		}
	} else {
		printUsage()
	}
}
