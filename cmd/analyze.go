package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/analyze"
	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AnalyzeCmd holds the analyze cmd flags
type AnalyzeCmd struct {
	Namespace string
	Wait      bool
}

// NewAnalyzeCmd creates a new login command
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
	analyzeCmd.Flags().BoolVar(&cmd.Wait, "wait", true, "Wait for pods to become running")

	return analyzeCmd
}

// RunAnalyze executes the functionality devspace analyze
func (cmd *AnalyzeCmd) RunAnalyze(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	namespace := "default"
	if configExists == true {
		// Configure cloud provider
		err = cloud.Configure(log.GetInstance())
		if err != nil {
			log.Fatalf("Unable to configure cloud provider: %v", err)
		}

		config := configutil.GetConfig()
		defaultNamespace, err := configutil.GetDefaultNamespace(config)
		if err != nil {
			log.Fatal(err)
		}

		namespace = defaultNamespace
	}
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	// Create kubectl client either from config or take the active current context
	var client *kubernetes.Clientset
	var config *rest.Config

	if configExists {
		config, err = kubectl.GetClientConfig()
		if err != nil {
			log.Fatal(err)
		}

		// Create kubectl client and switch context if specified
		client, err = kubectl.NewClient()
		if err != nil {
			log.Fatalf("Unable to create new kubectl client: %v", err)
		}
	} else {
		config, err = kubectl.GetClientConfigFromKubectl()
		if err != nil {
			log.Fatal(err)
		}

		client, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = analyze.Analyze(client, config, namespace, !cmd.Wait, log.GetInstance())
	if err != nil {
		log.Fatalf("Error during analyze: %v", err)
	}
}
