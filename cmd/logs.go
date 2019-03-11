package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// LogsCmd holds the logs cmd flags
type LogsCmd struct {
	selector          string
	namespace         string
	labelSelector     string
	container         string
	pod               string
	config            string
	pick              bool
	follow            bool
	lastAmountOfLines int
}

// NewLogsCmd creates a new login command
func NewLogsCmd() *cobra.Command {
	cmd := &LogsCmd{}

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Prints the logs of a pod and attaches to it",
		Long: `
#######################################################
#################### devspace logs ####################
#######################################################
Logs prints the last log of a pod container and attachs 
to it

Example:
devspace logs
devspace logs --namespace=mynamespace
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunLogs,
	}

	logsCmd.Flags().StringVarP(&cmd.selector, "selector", "s", "", "Selector name (in config) to select pod/container for terminal")
	logsCmd.Flags().StringVarP(&cmd.container, "container", "c", "", "Container name within pod where to execute command")
	logsCmd.Flags().StringVar(&cmd.pod, "pod", "", "Pod to print the logs of")
	logsCmd.Flags().StringVarP(&cmd.labelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	logsCmd.Flags().StringVarP(&cmd.namespace, "namespace", "n", "", "Namespace where to select pods")
	logsCmd.Flags().BoolVarP(&cmd.pick, "pick", "p", false, "Select a pod to stream logs from")
	logsCmd.Flags().BoolVarP(&cmd.follow, "follow", "f", false, "Attach to logs afterwards")
	logsCmd.Flags().IntVar(&cmd.lastAmountOfLines, "lines", 200, "Max amount of lines to print from the last log")
	logsCmd.Flags().StringVar(&cmd.config, "config", configutil.ConfigPath, "The devspace config file to load (default: '.devspace/config.yaml'")

	return logsCmd
}

// RunLogs executes the functionality devspace logs
func (cmd *LogsCmd) RunLogs(cobraCmd *cobra.Command, args []string) {
	// Set config root
	if configutil.ConfigPath != cmd.config {
		configutil.ConfigPath = cmd.config
	}

	// Set config root
	_, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	log.StartFileLogging()

	// Get kubectl client
	kubectl, err := kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	// Build params
	params := targetselector.CmdParameter{}
	if cmd.selector != "" {
		params.Selector = &cmd.selector
	}
	if cmd.container != "" {
		params.ContainerName = &cmd.container
	}
	if cmd.labelSelector != "" {
		params.LabelSelector = &cmd.labelSelector
	}
	if cmd.namespace != "" {
		params.Namespace = &cmd.namespace
	}
	if cmd.pod != "" {
		params.PodName = &cmd.pod
	}
	if cmd.pick != false {
		params.Pick = &cmd.pick
	}

	// Start terminal
	err = services.StartLogs(kubectl, params, cmd.follow, int64(cmd.lastAmountOfLines), log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}
}
