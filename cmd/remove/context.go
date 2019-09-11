package remove

import (
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type contextCmd struct {
	AllSpaces bool
	Provider  string
}

func newContextCmd() *cobra.Command {
	cmd := &contextCmd{}

	contextCmd := &cobra.Command{
		Use:   "context",
		Short: "Removes a kubectl-context",
		Long: `
#######################################################
############# devspace remove context #################
#######################################################
Removes a kubectl-context

Example:
devspace remove context myspace
devspace remove context --all-spaces
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemoveContext,
	}

	contextCmd.Flags().BoolVar(&cmd.AllSpaces, "all-spaces", false, "Remove all kubectl contexts belonging to Spaces")
	contextCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	return contextCmd
}

// RunRemoveContext executes the devspace remove context functionality
func (cmd *contextCmd) RunRemoveContext(cobraCmd *cobra.Command, args []string) {
	// Remove all contexts
	if cmd.AllSpaces {
		// Check if user has specified a certain provider
		var cloudProvider *string
		if cmd.Provider != "" {
			cloudProvider = &cmd.Provider
		}

		// Get provider
		provider, err := cloudpkg.GetProvider(cloudProvider, log.GetInstance())
		if err != nil {
			log.Fatalf("Error getting cloud context: %v", err)
		}

		// Retrieve spaces
		spaces, err := provider.GetSpaces()
		if err != nil {
			log.Fatal(err)
		}

		for _, space := range spaces {
			// Remove kube context
			err = cloudpkg.DeleteKubeContext(space)
			if err != nil {
				log.Fatalf("Error deleting kube context: %v", err)
			}

			log.Donef("Removed kubectl context for space %s", space.Name)
		}

		log.Done("All space kubectl contexts removed")
		return
	}

	// Load kube-config
	kubeConfig, err := kubeconfig.LoadRawConfig()
	if err != nil {
		log.Fatalf("Unable to load kube-config: %v", err)
	}

	var contextName string
	if len(args) > 0 {
		// First arg is context name
		contextName = args[0]

		if _, contextExists := kubeConfig.Contexts[contextName]; !contextExists {
			log.Fatalf("Kube-context '%s' does not exist", contextName)
		}
	} else {
		contexts := []string{}
		for ctx := range kubeConfig.Contexts {
			contexts = append(contexts, ctx)
		}

		contextName, err = survey.Question(&survey.QuestionOptions{
			Question: "Which context do you want to remove?",
			Options:  contexts,
		}, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}
	}

	oldCurrentContext := kubeConfig.CurrentContext

	// Remove the context
	err = kubeconfig.DeleteKubeContext(kubeConfig, contextName)
	if err != nil {
		log.Fatalf("Error remove context: %v", err)
	}

	// Save updated kube-config
	err = kubeconfig.SaveConfig(kubeConfig)
	if err != nil {
		log.Fatalf("Error saving kube file: %v", err)
	}

	if oldCurrentContext != kubeConfig.CurrentContext {
		log.Infof("Your kube-context has been updated to '%s'", ansi.Color(kubeConfig.CurrentContext, "white+b"))
	}

	log.Donef("Kube-context '%s' has been successfully removed", args[0])
}
