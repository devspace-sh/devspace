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
	fmt.Fprintf(os.Stderr, "Usage: sync [--version] [--upstream] [--downstream] [--exclude] PATH\n")
	os.Exit(1)
}

var version string

func main() {
	var (
		excludePaths arrayFlags

		isDownstream = flag.Bool("downstream", false, "Starts the downstream service")
		isUpstream   = flag.Bool("upstream", false, "Starts the upstream service")
		showVersion  = flag.Bool("version", false, "Shows the version")
	)

	flag.Var(&excludePaths, "exclude", "The exclude paths for downstream watching")
	flag.Parse()

	// Should we just print the version?
	if *showVersion {
		if version == "" {
			version = "latest"
		}

		fmt.Printf("%s", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		printUsage()
	}

	// we have to resolve the real local path, because the watcher gives us the real path always
	realLocalPath, err := filepath.EvalSymlinks(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	absolutePath, err := filepath.Abs(realLocalPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	if *isDownstream {
		err := server.StartDownstreamServer(absolutePath, excludePaths, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(1)
		}
	} else if *isUpstream {
		err := server.StartUpstreamServer(absolutePath, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(1)
		}
	} else {
		printUsage()
	}
}
