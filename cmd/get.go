package cmd

import (
	"os"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
	git "gopkg.in/src-d/go-git.v4"
)

// GetCmd is a struct that defines a command call for "get"
type GetCmd struct {
	flags *GetCmdFlags
}

// GetCmdFlags are the flags available for the get-command
type GetCmdFlags struct {
}

func init() {
	cmd := &EnterCmd{
		flags: &EnterCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a devspace project",
		Long: `
#######################################################
################### devspace get ######################
#######################################################
Clone a devspace project.

Example:

devspace get https://github.com/covexo/devspace-quickstart-nodejs
#######################################################`,
		Args: cobra.RangeArgs(1, 2),
		Run:  cmd.Run,
	}

	rootCmd.AddCommand(cobraCmd)
}

// Run executes the command logic
func (cmd *GetCmd) Run(cobraCmd *cobra.Command, args []string) {
	directoryName := "devspace"
	if len(args) == 2 {
		directoryName = args[1]
	}

	_, err := git.PlainClone(directoryName, false, &git.CloneOptions{
		URL:      args[0],
		Progress: os.Stdout,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chdir(directoryName)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully checked out %s into %s", args[0], directoryName)
}
