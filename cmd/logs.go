package cmd

import "github.com/spf13/cobra"

// LogsCmd holds the logs cmd flags
type LogsCmd struct {
	selector          string
	namespace         string
	labelSelector     string
	container         string
	config            string
	lastAmountOfLines int
	attach            bool
}

// NewLogsCmd creates a new login command
func NewLogsCmd() *cobra.Command {
	cmd := &LogsCmd{}

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Prints the logs of a pods and attaches to it",
		Long: `
	#######################################################
	#################### devspace logs ####################
	#######################################################
	Logs prints the last log of a pod container and attachs to it

	Example:
	devspace logs
	devspace analyze --namespace=mynamespace
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunLogs,
	}

	return logsCmd
}

// RunLogs executes the functionality devspace logs
func (cmd *LogsCmd) RunLogs(cobraCmd *cobra.Command, args []string) {

}
