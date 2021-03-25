package cmd

import (
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/analyze"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// AnalyzeCmd holds the analyze cmd flags
type AnalyzeCmd struct {
	*flags.GlobalFlags

	Wait              bool
	Patient           bool
	Timeout           int
	IgnorePodRestarts bool
}

// NewAnalyzeCmd creates a new analyze command
func NewAnalyzeCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &AnalyzeCmd{GlobalFlags: globalFlags}

	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyzes a kubernetes namespace and checks for potential problems",
		Long: `
#######################################################
################## devspace analyze ###################
#######################################################
Analyze checks a namespaces events, replicasets, services
and pods for potential problems

Example:
devspace analyze
devspace analyze --namespace=mynamespace
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAnalyze(f, plugins, cobraCmd, args)
		},
	}

	analyzeCmd.Flags().BoolVar(&cmd.Wait, "wait", true, "Wait for pods to get ready if they are just starting")
	analyzeCmd.Flags().IntVar(&cmd.Timeout, "timeout", 120, "Timeout until analyze should stop waiting")
	analyzeCmd.Flags().BoolVar(&cmd.Patient, "patient", false, "If true, analyze will ignore failing pods and events until every deployment, statefulset, replicaset and pods are ready or the timeout is reached")
	analyzeCmd.Flags().BoolVar(&cmd.IgnorePodRestarts, "ignore-pod-restarts", false, "If true, analyze will ignore the restart events of running pods")

	return analyzeCmd
}

// RunAnalyze executes the functionality "devspace analyze"
func (cmd *AnalyzeCmd) RunAnalyze(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}

	// Load generated config if possible
	var generatedConfig *generated.Config
	if configExists {
		generatedConfig, err = configLoader.LoadGenerated(cmd.ToConfigOptions())
		if err != nil {
			return err
		}
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, log)
	if err != nil {
		return err
	}

	// Create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return err
	}

	// Print warning
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, log)
	if err != nil {
		return err
	}

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "analyze", client.CurrentContext(), client.Namespace(), nil)
	if err != nil {
		return err
	}

	// Override namespace
	namespace := client.Namespace()
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	err = analyze.NewAnalyzer(client, log).Analyze(namespace, analyze.Options{
		Wait:              cmd.Wait,
		Timeout:           cmd.Timeout,
		Patient:           cmd.Patient,
		IgnorePodRestarts: cmd.IgnorePodRestarts,
	})
	if err != nil {
		return errors.Wrap(err, "analyze")
	}

	return nil
}
