package list

import (
	"sort"
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type contextsCmd struct{}

func newContextsCmd() *cobra.Command {
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
		RunE: cmd.RunListContexts,
	}

	return contextsCmd
}

// RunListContexts executes the functionality "devspace list contexts"
func (cmd *contextsCmd) RunListContexts(cobraCmd *cobra.Command, args []string) error {
	// Load kube-config
	kubeConfig, err := kubeconfig.LoadRawConfig()
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

	if defaultFound == false {
		contextRows = append(contextRows, []string{
			kubeConfig.CurrentContext,
			"true",
		})
	}

	log.PrintTable(log.GetInstance(), headerColumnNames, contextRows)
	return nil
}
