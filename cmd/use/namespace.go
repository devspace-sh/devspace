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
		Short: "Tells DevSpace which namespace to use",
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

	// Get default context
	client, err := kubectl.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	// Check if current kube context belongs to a space
	isSpace, err := kubeconfig.IsCloudSpace(client.CurrentContext)
	if err != nil {
		log.Fatalf("Unable to check if context belongs to Space: %v", err)
	}
	if isSpace {
		log.Fatalf("Current kube-context belongs to a Space created by DevSpace Cloud. Changing the default namespace for a Space context is not possible.")
	}

	// Load kube-config
	kubeConfig, err := kubeconfig.LoadRawConfig()
	if err != nil {
		log.Fatalf("Unable to load kube-config: %v", err)
	}

	if kubeConfig.Contexts[client.CurrentContext] == nil {
		log.Fatalf("Couldn't find kube context '%s' in kube config", client.CurrentContext)
	}

	// Remember current default namespace
	oldDefaultNamespace := kubeConfig.Contexts[client.CurrentContext].Namespace

	namespace := ""
	if len(args) > 0 {
		namespace = args[0]
	} else if !cmd.Reset {
		namespaceList, err := client.Client.CoreV1().Namespaces().List(metav1.ListOptions{})
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

	if oldDefaultNamespace != namespace {
		// Set namespace as default for used kube-context
		kubeConfig.Contexts[client.CurrentContext].Namespace = namespace

		// Save updated kube-config
		err = kubeconfig.SaveConfig(kubeConfig)
		if err != nil {
			log.Fatalf("Error saving kube config: %v")
		}

		log.Infof("The default namespace of your current kube-context '%s' has been updated to '%s'", ansi.Color(kubeConfig.CurrentContext, "white+b"), ansi.Color(namespace, "white+b"))
		log.Infof("\r         To revert this operation, run: %s\n", ansi.Color("devspace use namespace "+oldDefaultNamespace, "white+b"))
	}

	log.Donef("Successfully set default namespace to '%s'", ansi.Color(namespace, "white+b"))

	if configExists {
		// Get generated config
		generatedConfig, err := generated.LoadConfig("")
		if err != nil {
			log.Fatal(err)
		}

		// Reset namespace cache
		generatedConfig.GetActive().LastContext = nil

		// Save generated config
		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatal(err)
		}

		log.Infof("\r         \nRun:\n- `%s` to develop application\n- `%s` to deploy application\n", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
	}
}
