package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/sync/server"
)

func printUsage() {
	fmt.Printf("Usage: sync [--upstream] [--downstream] PATH\n")
	os.Exit(1)
}

func main() {
	isDownstream := flag.Bool("downstream", false, "Starts the downstream service")
	isUpstream := flag.Bool("upstream", false, "Starts the upstream service")

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		printUsage()
	}

	absolutePath, err := filepath.Abs(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	if *isDownstream {
		err := server.StartDownstreamServer(absolutePath, os.Stdin, os.Stdout)
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
