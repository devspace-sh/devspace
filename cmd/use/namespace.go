package use

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type namespaceCmd struct{}

func newNamespaceCmd() *cobra.Command {
	cmd := &namespaceCmd{}

	useNamespace := &cobra.Command{
		Use:   "namespace",
		Short: "Tells DevSpace which namespace to deploy to",
		Long: `
#######################################################
################ devspace use space ###################
#######################################################
Set the default namesapce to deploy to

Example:
devspace use namespace my-space
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunUseNamespace,
	}

	return useNamespace
}

// RunUseNamespace executes the functionality "devspace use namespace"
func (cmd *namespaceCmd) RunUseNamespace(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	if !configExists {
		log.Fatal("Unable to find DevSpace config")
	}

	// First arg is namespace name
	namespace := args[0]

	// Get current kubectl context
	kubeConfig, err := kubeconfig.LoadRawConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Get generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Make sure DevSpace is not using a space anymore
	generatedConfig.CloudSpace = nil

	// Configure DevSpace to use plain namespace
	generatedConfig.Namespace = &generated.NamespaceConfig{
		Name:        &namespace,
		KubeContext: &kubeConfig.CurrentContext,
	}

	// Save generated config
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully configured DevSpace to use namespace %s", namespace)
	log.Infof("\r          \nRun:\n- `%s` to develop application\n- `%s` to deploy application\n", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
}
