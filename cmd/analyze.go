package cmd

import (
	"context"

	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// AnalyzeCmd holds the analyze cmd flags
type AnalyzeCmd struct {
	Namespace   string
	KubeContext string

	Wait bool
}

// NewAnalyzeCmd creates a new analyze command
func NewAnalyzeCmd() *cobra.Command {
	cmd := &AnalyzeCmd{}

	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyzes a kubernetes namespace and checks for potential problems",
		Long: `
#######################################################
################## devspace analyze ###################
#######################################################
Analyze checks a namespaces events, replicasets, services
and pods for potential problems

Example:
devspace analyze
devspace analyze --namespace=mynamespace
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunAnalyze,
	}

	analyzeCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "The kubernetes namespace to analyze")
	analyzeCmd.Flags().StringVar(&cmd.KubeContext, "kube-context", "", "The kubernetes context to use")

	analyzeCmd.Flags().BoolVar(&cmd.Wait, "wait", true, "Wait for pods to get ready if they are just starting")

	return analyzeCmd
}

// RunAnalyze executes the functionality "devspace analyze"
func (cmd *AnalyzeCmd) RunAnalyze(cobraCmd *cobra.Command, args []string) {
	// Set config root
	_, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	// Create kubectl client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, false)
	if err != nil {
		log.Fatal(err)
	}

	// Print warning
	err = client.PrintWarning(context.Background(), false, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Signal that we are working on the space if there is any
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Override namespace
	namespace := client.Namespace
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	err = analyze.Analyze(client, namespace, !cmd.Wait, log.GetInstance())
	if err != nil {
		log.Fatalf("Error during analyze: %v", err)
	}
}
