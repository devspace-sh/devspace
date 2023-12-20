package cleanup

import (
	"context"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/build/localregistry"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type localRegistryCmd struct {
	*flags.GlobalFlags
}

func newLocalRegistryCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &localRegistryCmd{GlobalFlags: globalFlags}

	localRegistryCmd := &cobra.Command{
		Use:   "local-registry",
		Short: "Deletes the local image registry",
		Long: ` 
#######################################################
######### devspace cleanup local-registry #############
#######################################################
Deletes the local image registry
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunCleanupLocalRegistry(f, cobraCmd, args)
		}}

	return localRegistryCmd
}

// RunCleanupLocalRegistry executes the cleanup local-registry command logic
func (cmd *localRegistryCmd) RunCleanupLocalRegistry(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	ctx := context.Background()
	log := f.GetLog()

	// set config root
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		log.Warnf("Unable to create new kubectl client: %v", err)
		log.WriteString(logrus.WarnLevel, "\n")
		client = nil
	}

	// load generated config
	localCache, err := configLoader.LoadLocalCache()
	if err != nil {
		return errors.Errorf("error loading local cache: %v", err)
	}

	if client != nil {
		// If the current kube context or namespace is different than old,
		// show warnings and reset kube client if necessary
		client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, log)
		if err != nil {
			return err
		}
	}

	// load config
	configInterface, err := configLoader.LoadWithCache(ctx, localCache, client, cmd.ToConfigOptions(), log)
	if err != nil {
		return err
	}

	// clean up registry according to options
	config := configInterface.Config()
	options := localregistry.NewDefaultOptions().
		WithNamespace(client.Namespace()).
		WithLocalRegistryConfig(config.LocalRegistry)

	hasStatefulSet := true
	_, err = client.KubeClient().AppsV1().StatefulSets(options.Namespace).Get(ctx, options.Name, v1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			hasStatefulSet = false
		} else {
			return errors.Wrap(err, "clean up statefulset")
		}
	}

	hasDeployment := true
	_, err = client.KubeClient().AppsV1().Deployments(options.Namespace).Get(ctx, options.Name, v1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			hasDeployment = false
		} else {
			return errors.Wrap(err, "clean up deployment")
		}
	}

	hasService := true
	_, err = client.KubeClient().CoreV1().Services(options.Namespace).Get(ctx, options.Name, v1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			hasService = false
		} else {
			return errors.Wrap(err, "clean up service")
		}
	}

	if !hasStatefulSet && !hasDeployment && !hasService {
		log.Donef("No local registry found.")
		return nil
	}

	// prompt user since this is a destructive action
	cleanupAnswer, err := log.Question(&survey.QuestionOptions{
		Question: "This will delete your local registry and all the images it contains. Do you wish to continue?",
		Options: []string{
			"Yes",
			"No",
		},
	})
	if err != nil {
		return err
	}

	if cleanupAnswer == "No" {
		return nil
	}

	if hasStatefulSet {
		err = client.KubeClient().AppsV1().StatefulSets(options.Namespace).Delete(ctx, options.Name, v1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return errors.Wrap(err, "clean up statefulset")
		}
	}

	if hasDeployment {
		err = client.KubeClient().AppsV1().Deployments(options.Namespace).Delete(ctx, options.Name, v1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return errors.Wrap(err, "clean up deployment")
		}
	}

	if hasService {
		err = client.KubeClient().CoreV1().Services(options.Namespace).Delete(ctx, options.Name, v1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return errors.Wrap(err, "clean up service")
		}
	}

	log.Donef("Successfully cleaned up local registry")
	return nil
}
