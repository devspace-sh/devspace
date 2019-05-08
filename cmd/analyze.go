package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// AnalyzeCmd holds the analyze cmd flags
type AnalyzeCmd struct {
	Namespace string
	Wait      bool
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
	analyzeCmd.Flags().BoolVar(&cmd.Wait, "wait", true, "Wait for pods to get ready if they are just starting")

	return analyzeCmd
}

// RunAnalyze executes the functionality "devspace analyze"
func (cmd *AnalyzeCmd) RunAnalyze(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	var devSpaceConfig *latest.Config
	if configExists {
		devSpaceConfig = configutil.GetConfig()
	}

	// Create kubectl client
	config, err := kubectl.GetClientConfig(devSpaceConfig)
	if err != nil {
		log.Fatal(err)
	}

	namespace := ""
	if configExists == true {
		config := configutil.GetConfig()

		namespace, err = configutil.GetDefaultNamespace(config)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		namespace, err = configutil.GetDefaultNamespace(nil)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Override namespace
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	err = analyze.Analyze(config, namespace, !cmd.Wait, log.GetInstance())
	if err != nil {
		log.Fatalf("Error during analyze: %v", err)
	}
}
