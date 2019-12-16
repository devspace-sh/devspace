package list

import (
	"strconv"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type namespacesCmd struct {
	*flags.GlobalFlags
}

func newNamespacesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &namespacesCmd{GlobalFlags: globalFlags}

	namespacesCmd := &cobra.Command{
		Use:   "namespaces",
		Short: "Lists all namespaces in the current context",
		Long: `
#######################################################
############ devspace list namespaces #################
#######################################################
Lists all namespaces in the selected kube context
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: cmd.RunListNamespaces,
	}

	return namespacesCmd
}

// RunListNamespaces runs the list namespaces command logic
func (cmd *namespacesCmd) RunListNamespaces(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log.GetInstance())
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}

	// Load generated config if possible
	var generatedConfig *generated.Config
	if configExists {
		generatedConfig, err = configLoader.Generated()
		if err != nil {
			return err
		}
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, log.GetInstance())
	if err != nil {
		return err
	}

	// Get kubectl client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	namespaces, err := client.KubeClient().CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "list namespaces")
	}

	headerColumnNames := []string{
		"Name",
		"Default",
		"Exists",
	}

	// Transform values into string arrays
	namespaceRows := make([][]string, 0, len(namespaces.Items))
	defaultFound := false
	for _, namespace := range namespaces.Items {
		namespaceRows = append(namespaceRows, []string{
			namespace.Name,
			strconv.FormatBool(namespace.Name == client.Namespace()),
			"true",
		})

		if namespace.Name == client.Namespace() {
			defaultFound = true
		}
	}

	if defaultFound == false {
		namespaceRows = append(namespaceRows, []string{
			client.Namespace(),
			"true",
			"false",
		})
	}

	log.PrintTable(log.GetInstance(), headerColumnNames, namespaceRows)
	return nil
}
