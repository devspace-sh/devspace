package remove

import (
	"sort"

	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type contextCmd struct {
	AllSpaces bool
	Provider  string
}

func newContextCmd(f factory.Factory) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunRemoveContext(f, cobraCmd, args)
		}}

	contextCmd.Flags().BoolVar(&cmd.AllSpaces, "all-spaces", false, "Remove all kubectl contexts belonging to Spaces")
	contextCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	return contextCmd
}

// RunRemoveContext executes the devspace remove context functionality
func (cmd *contextCmd) RunRemoveContext(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	log := f.GetLog()
	kubeLoader := f.NewKubeConfigLoader()
	// Remove all contexts
	if cmd.AllSpaces {
		// Get provider
		provider, err := f.GetProvider(cmd.Provider, log)
		if err != nil {
			return errors.Wrap(err, "log into provider")
		}

		// Retrieve spaces
		spaces, err := provider.Client().GetSpaces()
		if err != nil {
			return err
		}

		for _, space := range spaces {
			// Remove kube context
			err = provider.DeleteKubeContext(space)
			if err != nil {
				return errors.Wrap(err, "delete kube context")
			}

			log.Donef("Removed kubectl context for space %s", space.Name)
		}

		log.Done("All space kubectl contexts removed")
		return nil
	}

	// Load kube-config
	kubeConfig, err := kubeLoader.LoadRawConfig()
	if err != nil {
		return errors.Wrap(err, "load kube config")
	}

	var contextName string
	if len(args) > 0 {
		// First arg is context name
		contextName = args[0]

		if _, contextExists := kubeConfig.Contexts[contextName]; !contextExists {
			return errors.Errorf("Kube-context '%s' does not exist", contextName)
		}
	} else {
		contexts := []string{}
		for ctx := range kubeConfig.Contexts {
			contexts = append(contexts, ctx)
		}

		sort.Strings(contexts)

		contextName, err = log.Question(&survey.QuestionOptions{
			Question: "Which context do you want to remove?",
			Options:  contexts,
		})
		if err != nil {
			return err
		}
	}

	oldCurrentContext := kubeConfig.CurrentContext

	// Remove the context
	err = kubeLoader.DeleteKubeContext(kubeConfig, contextName)
	if err != nil {
		return errors.Wrap(err, "delete kube context")
	}

	// Save updated kube-config
	err = kubeLoader.SaveConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "save kube config")
	}

	if oldCurrentContext != kubeConfig.CurrentContext {
		log.Infof("Your kube-context has been updated to '%s'", ansi.Color(kubeConfig.CurrentContext, "white+b"))
	}

	log.Donef("Kube-context '%s' has been successfully removed", contextName)
	return nil
}
