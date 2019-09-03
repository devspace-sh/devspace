package use

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type namespaceCmd struct {
	Reset bool
}

func newNamespaceCmd() *cobra.Command {
	cmd := &namespaceCmd{}

	useNamespace := &cobra.Command{
		Use:   "namespace",
		Short: "Tells DevSpace which namespace to deploy to",
		Long: `
#######################################################
############## devspace use namespace #################
#######################################################
Set the default namespace to deploy to

Example:
devspace use namespace my-namespace
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunUseNamespace,
	}

	useNamespace.Flags().BoolVar(&cmd.Reset, "reset", false, "Resets the default namespace of the current kube-context")

	return useNamespace
}

// RunUseNamespace executes the functionality "devspace use namespace"
func (cmd *namespaceCmd) RunUseNamespace(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	// Load kube-config
	kubeConfig, err := kubeconfig.LoadRawConfig()
	if err != nil {
		log.Fatalf("Unable to load kube-config: %v", err)
	}

	// Get current kube-context
	currentContext, ok := kubeConfig.Contexts[kubeConfig.CurrentContext]
	if !ok {
		log.Fatalf("Unable to find current kube-context '%s' in kube-config file", kubeConfig.CurrentContext)
	}

	// Check if current kube-context blongs to Space
	isSpace, err := kubeconfig.IsCloudSpace(currentContext)
	if err != nil {
		log.Fatalf("Unable to check if context belongs to Space: %v", err)
	}

	if isSpace {
		log.Fatalf("Current kube-context belongs to a Space created by DevSpace Cloud. Changing the default namespace for a Space context is not possible.")
	}

	// Remember current default namespace
	oldDefaultNamespace := currentContext.Namespace

	namespace := ""

	if len(args) > 0 {
		// First arg is namespace name
		namespace := args[0]

		if namespace == metav1.NamespaceDefault {
			log.Warn("Using the 'default' namespace of your cluster is highly discouraged as this namespace cannot be deleted.")
		}
	} else if !cmd.Reset {
		// Get kubernetes client
		client, err := kubectl.NewClient(nil)
		if err != nil {
			log.Fatal(err)
		}

		namespaceList, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Unable to list namespaces: %v", err)
		}

		namespaces := []string{}

		for _, ns := range namespaceList.Items {
			namespaces = append(namespaces, ns.Name)
		}

		namespace = survey.Question(&survey.QuestionOptions{
			Question: "Which namespace do you want to use?",
			Options:  namespaces,
		})
	}

	// Set namespace as default for current kube-context
	currentContext.Namespace = namespace

	if oldDefaultNamespace != namespace {
		// Save updated kube-config
		kubeconfig.SaveConfig(kubeConfig)

		log.Infof("The default namespace of your current kube-context '%s' has been updated to '%s'", ansi.Color(kubeConfig.CurrentContext, "white+b"), ansi.Color(namespace, "white+b"))
		log.Infof("\r          To revert this operation, run: %s", ansi.Color("devspace use namespace "+oldDefaultNamespace, "white+b"))
	}

	log.Donef("Successfully set default namespace to '%s'", ansi.Color(namespace, "white+b"))

	if configExists {
		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		// Reset namespace cache
		generatedConfig.Namespace = nil

		// Save generated config
		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatal(err)
		}

		log.Infof("\r          \nRun:\n- `%s` to develop application\n- `%s` to deploy application\n", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
	}
}
