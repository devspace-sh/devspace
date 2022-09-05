package list

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"strconv"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type namespacesCmd struct {
	*flags.GlobalFlags
}

func newNamespacesCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
		// Args: cobra.NoArgs,
		DisableFlagParsing: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunListNamespaces(f, cobraCmd, args)
		}}

	return namespacesCmd
}

// RunListNamespaces runs the list namespaces command logic
func (cmd *namespacesCmd) RunListNamespaces(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	logger := f.GetLog()
	// Set config root
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}

	// Get kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	// Load generated config if possible
	var localCache localcache.Cache
	if configExists {
		localCache, err = configLoader.LoadLocalCache()
		if err != nil {
			return err
		}
	}

	// If the current kube context or namespace is different from old,
	// show warnings and reset kube client if necessary
	client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, logger)
	if err != nil {
		return err
	}

	namespaces, err := client.KubeClient().CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
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

	if !defaultFound {
		namespaceRows = append(namespaceRows, []string{
			client.Namespace(),
			"true",
			"false",
		})
	}

	log.PrintTable(logger, headerColumnNames, namespaceRows)
	return nil
}
