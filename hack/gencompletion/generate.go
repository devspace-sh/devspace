package main

import (
	"fmt"
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
)

func main() {
	// create a new factory
	f := factory.DefaultFactory()

	// build the root command
	rootCmd := cmd.BuildRoot(f)

	// generate the completions
	err := rootCmd.GenBashCompletionFile("completion/bash.sh")
	if err != nil {
		fmt.Println(err)
	}

	err = rootCmd.GenZshCompletionFile("completion/zsh-completion")
	if err != nil {
		fmt.Println(err)
	}

	err = rootCmd.GenPowerShellCompletionFile("completion/powershell.ps")
	if err != nil {
		fmt.Println(err)
	}

	err = rootCmd.GenFishCompletionFile("completion/fish.fish", true)
	if err != nil {
		fmt.Println(err)
	}

}
