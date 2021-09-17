package list

import (
	"sort"
	"strconv"

	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type contextsCmd struct{}

func newContextsCmd(f factory.Factory) *cobra.Command {
	cmd := &contextsCmd{}

	contextsCmd := &cobra.Command{
		Use:   "contexts",
		Short: "Lists all kube contexts",
		Long: `
#######################################################
############## devspace list contexts #################
#######################################################
Lists all available kube contexts

Example:
devspace list contexts
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListContexts(f, cobraCmd, args)
		}}

	return contextsCmd
}

// RunListContexts executes the functionality "devspace list contexts"
func (cmd *contextsCmd) RunListContexts(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	kubeLoader := f.NewKubeConfigLoader()
	// Load kube-config
	kubeConfig, err := kubeLoader.LoadRawConfig()
	if err != nil {
		return errors.Wrap(err, "load kube config")
	}

	headerColumnNames := []string{
		"Name",
		"Active",
	}

	contexts := []string{}
	for ctx := range kubeConfig.Contexts {
		contexts = append(contexts, ctx)
	}

	sort.Strings(contexts)

	contextRows := make([][]string, 0, len(contexts))
	defaultFound := false
	for _, context := range contexts {
		contextRows = append(contextRows, []string{
			context,
			strconv.FormatBool(context == kubeConfig.CurrentContext),
		})

		if context == kubeConfig.CurrentContext {
			defaultFound = true
		}
	}

	if !defaultFound {
		contextRows = append(contextRows, []string{
			kubeConfig.CurrentContext,
			"true",
		})
	}

	log.PrintTable(logger, headerColumnNames, contextRows)
	return nil
}
