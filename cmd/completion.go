package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	longDescription = `
	Outputs shell completion for the given shell (bash or zsh)

	This depends on the bash-completion binary.  Example installation instructions:
	OS X:
		$ brew install bash-completion
		$ source $(brew --prefix)/etc/bash_completion
		$ devspace completion bash > ~/.devspace-completion  # for bash users
		$ devspace completion fish > ~/.devspace-completion  # for fish users
		$ devspace completion zsh > ~/.devspace-completion   # for zsh users
		$ source ~/.devspace-completion
	Ubuntu:
		$ apt-get install bash-completion
		$ source /etc/bash-completion
		$ source <(devspace completion bash) # for bash users
		$ devspace completion fish | source # for fish users
		$ source <(devspace completion zsh)  # for zsh users

	Additionally, you may want to output the completion to a file and source in your .bashrc
`

	zshCompdef = "\ncompdef _devspace devspace\n"
)

// NewCompletionCmd returns the cobra command that outputs shell completion code
func NewCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use: "completion SHELL",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("missing shell: %s", strings.Join(cmd.ValidArgs, ", "))
			}
			return cobra.OnlyValidArgs(cmd, args)
		},
		ValidArgs: []string{"bash", "fish", "zsh"},
		Short:     "Outputs shell completion for the given shell (bash or zsh)",
		Long:      longDescription,
		RunE:      completion,
	}
}

func completion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return rootCmd(cmd).GenBashCompletion(os.Stdout)
	case "fish":
		return rootCmd(cmd).GenFishCompletion(os.Stdout, true)
	case "zsh":
		err := rootCmd(cmd).GenZshCompletion(os.Stdout)
		if err != nil {
			return err
		}
		_, err = io.WriteString(os.Stdout, zshCompdef)
		if err != nil {
			return err
		}
	}
	return nil
}

func rootCmd(cmd *cobra.Command) *cobra.Command {
	parent := cmd
	for parent.HasParent() {
		parent = parent.Parent()
	}
	return parent
}
