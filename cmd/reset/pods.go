package reset

import (
	"context"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	dependencytypes "github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type podsCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

func newPodsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &podsCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	podsCmd := &cobra.Command{
		Use:   "pods",
		Short: "Resets the replaced pods",
		Long: `
#######################################################
############### devspace reset pods ###################
#######################################################
Resets the replaced pods to its original state

Examples:
devspace reset pods
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunResetPods(f, cobraCmd, args)
		}}

	return podsCmd
}

// RunResetPods executes the reset pods command logic
func (cmd *podsCmd) RunResetPods(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	cmd.log = f.GetLog()
	configOptions := cmd.ToConfigOptions(cmd.log)
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Get config with adjusted cluster config
	generatedConfig, err := configLoader.LoadGenerated(configOptions)
	if err != nil {
		return err
	}
	configOptions.GeneratedConfig = generatedConfig

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}
	configOptions.KubeClient = client

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, cmd.log)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	configInterface, err := configLoader.Load(configOptions, cmd.log)
	if err != nil {
		return err
	}

	// Resolve dependencies
	dependencies, err := f.NewDependencyManager(configInterface, client, configOptions, cmd.log).ResolveAll(dependency.ResolveOptions{
		UpdateDependencies: false,
		Silent:             true,
		Verbose:            false,
	})
	if err != nil {
		cmd.log.Warnf("Error resolving dependencies: %v", err)
	}
	for _, dep := range dependencies {
		if dep.DependencyConfig().Dev != nil && dep.DependencyConfig().Dev.ReplacePods && len(dep.Config().Config().Dev.ReplacePods) > 0 {
			ResetPods(client, dep.Config(), dep.Children(), cmd.log)
		}
	}

	// reset the pods
	ResetPods(client, configInterface, dependencies, cmd.log)
	return nil
}

// ResetPods deletes the pods created by dev.replacePods
func ResetPods(client kubectl.Client, config config.Config, dependencies []dependencytypes.Dependency, log log.Logger) {
	// create pod replacer
	resetted := 0
	errored := false
	podReplacer := podreplace.NewPodReplacer()
	for _, replacePod := range config.Config().Dev.ReplacePods {
		deletedPod, err := podReplacer.RevertReplacePod(context.TODO(), client, config, dependencies, replacePod, log)
		if err != nil {
			errored = true
			log.Warnf("Error reverting replaced pod: %v", err)
		} else if deletedPod != nil {
			resetted++
		}
	}

	if resetted == 0 {
		if !errored {
			log.Info("No pods to reset found")
		}
	} else {
		log.Donef("Successfully reset %d pods", resetted)
	}
}
