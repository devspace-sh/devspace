package use

import (
	"context"

	"github.com/loft-sh/devspace/cmd/flags"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type namespaceCmd struct {
	*flags.GlobalFlags
	Reset  bool
	Create bool
}

func newNamespaceCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &namespaceCmd{GlobalFlags: globalFlags}

	useNamespace := &cobra.Command{
		Use:   "namespace",
		Short: "Tells DevSpace which namespace to use",
		Long: `
#######################################################
############## devspace use namespace #################
#######################################################
Sets the default namespace to deploy to

Example:
devspace use namespace my-namespace
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunUseNamespace(f, cobraCmd, args)
		},
	}

	useNamespace.Flags().BoolVar(&cmd.Reset, "reset", false, "Resets the default namespace of the current kube-context")
	useNamespace.Flags().BoolVar(&cmd.Create, "create", false, "Create the namespace if it doesn't exist")

	return useNamespace
}

// RunUseNamespace executes the functionality "devspace use namespace"
func (cmd *namespaceCmd) RunUseNamespace(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Get default context
	log := f.GetLog()
	client, err := f.NewKubeDefaultClient()
	if err != nil {
		return err
	}

	// Check if current kube context belongs to a space
	kubeLoader := client.KubeConfigLoader()

	// Load kube-config
	kubeConfig, err := kubeLoader.LoadRawConfig()
	if err != nil {
		return errors.Errorf("Unable to load kube-config: %v", err)
	}

	if kubeConfig.Contexts[client.CurrentContext()] == nil {
		return errors.Errorf("Couldn't find kube context '%s' in kube config", client.CurrentContext())
	}

	// Remember current default namespace
	oldDefaultNamespace := kubeConfig.Contexts[client.CurrentContext()].Namespace

	namespace := ""
	if len(args) > 0 {
		namespace = args[0]
		if cmd.Create {
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			_, err := client.KubeClient().CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			if err != nil && !kerrors.IsAlreadyExists(err) {
				return errors.Errorf("Unable to create namespace: %v", err)
			}
		}
	} else if !cmd.Reset {
		namespaceList, err := client.KubeClient().CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return errors.Errorf("Unable to list namespaces: %v", err)
		}

		namespaces := []string{}
		for _, ns := range namespaceList.Items {
			namespaces = append(namespaces, ns.Name)
		}

		namespace, err = log.Question(&survey.QuestionOptions{
			Question: "Which namespace do you want to use?",
			Options:  namespaces,
			Sort:     true,
		})
		if err != nil {
			return err
		}
	}

	if oldDefaultNamespace != namespace {
		// Set namespace as default for used kube-context
		kubeConfig.Contexts[client.CurrentContext()].Namespace = namespace

		// Save updated kube-config
		err = kubeLoader.SaveConfig(kubeConfig)
		if err != nil {
			return errors.Errorf("Error saving kube config: %v", err)
		}

		log.Infof("The default namespace of your current kube-context '%s' has been updated to '%s'", ansi.Color(kubeConfig.CurrentContext, "white+b"), ansi.Color(namespace, "white+b"))
		log.Infof("\r         To revert this operation, run: %s\n", ansi.Color("devspace use namespace "+oldDefaultNamespace, "white+b"))
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

	log.Donef("Successfully set default namespace to '%s'", ansi.Color(namespace, "white+b"))
	return nil
}
