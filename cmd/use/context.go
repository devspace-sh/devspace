package use

import (
	"fmt"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ContextCmd struct {
	*flags.GlobalFlags
}

func newContextCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ContextCmd{GlobalFlags: globalFlags}

	useContext := &cobra.Command{
		Use:   "context",
		Short: "Tells DevSpace which kube context to use",
		Long: `
#######################################################
############### devspace use context ##################
#######################################################
Switches the current kube-context

Example:
devspace use context my-context
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunUseContext(f, args)
		},
	}

	return useContext
}

// RunUseContext executes the functionality "devspace use namespace"
func (cmd *ContextCmd) RunUseContext(f factory.Factory, args []string) error {
	// Load kube-config
	log := f.GetLog()
	kubeLoader := f.NewKubeConfigLoader()
	kubeConfig, err := kubeLoader.LoadRawConfig()
	if err != nil {
		return errors.Wrap(err, "load kube config")
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

		context, err = log.Question(&survey.QuestionOptions{
			Question:     "Which context do you want to use?",
			DefaultValue: kubeConfig.CurrentContext,
			Options:      contexts,
			Sort:         true,
		})
		if err != nil {
			return err
		}
	}

	// check if context exists
	_, found := kubeConfig.Contexts[context]
	if !found {
		return fmt.Errorf("couldn't find context %s in kube config", context)
	}

	// Save old context
	oldContext := kubeConfig.CurrentContext

	// Set current kube-context
	kubeConfig.CurrentContext = context

	if oldContext != context {
		// Save updated kube-config
		_ = kubeLoader.SaveConfig(kubeConfig)

		log.Infof("Your kube-context has been updated to '%s'", ansi.Color(kubeConfig.CurrentContext, "white+b"))
		log.Infof("\r         To revert this operation, run: %s\n", ansi.Color("devspace use context "+oldContext, "white+b"))
	}

	// clear project kube context
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	err = ClearProjectKubeContext(configLoader, log)
	if err != nil {
		return errors.Wrap(err, "clear generated kube context")
	}

	log.Donef("Successfully set kube-context to '%s'", ansi.Color(context, "white+b"))
	return nil
}

func ClearProjectKubeContext(configLoader loader.ConfigLoader, log log.Logger) error {
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	} else if !configExists {
		return nil
	}

	// load config if it exists
	localCache, err := configLoader.LoadLocalCache()
	if err != nil {
		return err
	}

	// update last context
	localCache.SetLastContext(nil)

	// save it
	return localCache.Save()
}
