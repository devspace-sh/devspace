package cmd

import (
	"github.com/spf13/cobra"
)

// RunStatusSync executes the devspace status sync commad logic
func (cmd *StatusCmd) RunStatusSync(cobraCmd *cobra.Command, args []string) {
	loadConfig(&cmd.workdir, &cmd.privateConfig, &cmd.dsConfig)

}
