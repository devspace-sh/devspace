package use

import (
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
		Short: "Tells DevSpace which context to use",
		Long: `
#######################################################
############### devspace use context ##################
#######################################################
Set the default context to deploy to

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

		for ctx, _ := range kubeConfig.Contexts {
			contexts = append(contexts, ctx)
		}

		context = survey.Question(&survey.QuestionOptions{
			Question: "Which context do you want to use?",
			Options:  contexts,
		})
	}
	oldContext := kubeConfig.CurrentContext

	// Set current kube-context
	kubeConfig.CurrentContext = context

	if oldContext != context {
		// Save updated kube-config
		kubeconfig.SaveConfig(kubeConfig)

		log.Infof("Your kube-context has been updated to '%s'", ansi.Color(kubeConfig.CurrentContext, "white+b"))
		log.Infof("\r          To revert this operation, run: %s\n", ansi.Color("devspace use context "+oldContext, "white+b"))
	}

	log.Donef("Successfully set kube-context to '%s'", ansi.Color(context, "white+b"))
}
