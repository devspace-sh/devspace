package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// LogsCmd holds the logs cmd flags
type LogsCmd struct {
	Selector          string
	Namespace         string
	LabelSelector     string
	Container         string
	Pod               string
	Pick              bool
	Follow            bool
	LastAmountOfLines int
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

	logsCmd.Flags().StringVarP(&cmd.Selector, "selector", "s", "", "Selector name (in config) to select pod/container for terminal")
	logsCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	logsCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to print the logs of")
	logsCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	logsCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "Namespace where to select pods")
	logsCmd.Flags().BoolVarP(&cmd.Pick, "pick", "p", false, "Select a pod")
	logsCmd.Flags().BoolVarP(&cmd.Follow, "follow", "f", false, "Attach to logs afterwards")
	logsCmd.Flags().IntVar(&cmd.LastAmountOfLines, "lines", 200, "Max amount of lines to print from the last log")

	return logsCmd
}

// RunLogs executes the functionality devspace logs
func (cmd *LogsCmd) RunLogs(cobraCmd *cobra.Command, args []string) {
	// Set config root
	_, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	var config *latest.Config
	if configutil.ConfigExists() {
		// Load generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading generated.yaml: %v", err)
		}

		// Get config with adjusted cluster config
		config, err := configutil.GetContextAjustedConfig(cmd.Namespace, "")
		if err != nil {
			log.Fatal(err)
		}

		// Signal that we are working on the space if there is any
		err = cloud.ResumeSpace(config, generatedConfig, true, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}
	}

	// Get kubectl client
	kubectl, err := kubectl.NewClient(config)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	// Build params
	params := targetselector.CmdParameter{}
	if cmd.Selector != "" {
		params.Selector = &cmd.Selector
	}
	if cmd.Container != "" {
		params.ContainerName = &cmd.Container
	}
	if cmd.LabelSelector != "" {
		params.LabelSelector = &cmd.LabelSelector
	}
	if cmd.Namespace != "" {
		params.Namespace = &cmd.Namespace
	}
	if cmd.Pod != "" {
		params.PodName = &cmd.Pod
	}
	if cmd.Pick != false {
		params.Pick = &cmd.Pick
	}

	// Start terminal
	err = services.StartLogs(config, kubectl, params, cmd.Follow, int64(cmd.LastAmountOfLines), log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}
}
