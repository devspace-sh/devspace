package use

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type contextCmd struct{}

func newContextCmd() *cobra.Command {
	cmd := &contextCmd{}

	useContext := &cobra.Command{
		Use:   "context",
		Short: "Tells DevSpace which kube context to use",
		Long: `
#######################################################
############### devspace use context ##################
#######################################################
Switch the current kube context

Example:
devspace use context my-context
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunUseContext,
	}

	return useContext
}

// RunUseContext executes the functionality "devspace use namespace"
func (cmd *contextCmd) RunUseContext(cobraCmd *cobra.Command, args []string) {
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

	var context string
	if len(args) > 0 {
		// First arg is context name
		context = args[0]
	} else {
		contexts := []string{}
		for ctx := range kubeConfig.Contexts {
			contexts = append(contexts, ctx)
		}

		context = survey.Question(&survey.QuestionOptions{
			Question: "Which context do you want to use?",
			Options:  contexts,
		})
	}

	// Save old context
	oldContext := kubeConfig.CurrentContext

	// Set current kube-context
	kubeConfig.CurrentContext = context

	if oldContext != context {
		// Save updated kube-config
		kubeconfig.SaveConfig(kubeConfig)

		log.Infof("Your kube-context has been updated to '%s'", ansi.Color(kubeConfig.CurrentContext, "white+b"))
		log.Infof("\r         To revert this operation, run: %s\n", ansi.Color("devspace use context "+oldContext, "white+b"))

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
		}
	}

	log.Donef("Successfully set kube-context to '%s'", ansi.Color(context, "white+b"))
}
