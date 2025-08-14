package cmd

import (
	"context"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/logs"
	"os"
	"time"

	"github.com/loft-sh/devspace/cmd/flags"
	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// LogsCmd holds the logs cmd flags
type LogsCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	ImageSelector string
	Image         string
	Container     string
	Pod           string
	Pick          bool

	Follow            bool
	Wait              bool
	LastAmountOfLines int
}

// NewLogsCmd creates a new login command
func NewLogsCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &LogsCmd{GlobalFlags: globalFlags}

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Prints the logs of a pod and attaches to it",
		Long: `
#######################################################
#################### devspace logs ####################
#######################################################
Prints the last log of a pod container and attachs 
to it

Example:
devspace logs
devspace logs --namespace=mynamespace
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.RunLogs(f)
		},
	}

	logsCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	logsCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to print the logs of")
	logsCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	logsCmd.Flags().StringVar(&cmd.ImageSelector, "image-selector", "", "The image to search a pod for (e.g. nginx, nginx:latest, ${runtime.images.app}, nginx:${runtime.images.app.tag})")
	logsCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod")
	logsCmd.Flags().BoolVarP(&cmd.Follow, "follow", "f", false, "Attach to logs afterwards")
	logsCmd.Flags().IntVar(&cmd.LastAmountOfLines, "lines", 200, "Max amount of lines to print from the last log")
	logsCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "Wait for the pod(s) to start if they are not running")

	return logsCmd
}

// RunLogs executes the functionality devspace logs
func (cmd *LogsCmd) RunLogs(f factory.Factory) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}

	// Get kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	// Load generated config if possible
	if configExists {
		localCache, err := configLoader.LoadLocalCache()
		if err != nil {
			return err
		}

		// If the current kube context or namespace is different from old,
		// show warnings and reset kube client if necessary
		client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, log)
		if err != nil {
			return err
		}
	}

	// create the context
	ctx := devspacecontext.NewContext(context.Background(), nil, log).WithKubeClient(client)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "logs")
	if err != nil {
		return err
	}

	// get image selector if specified
	imageSelector, err := getImageSelector(ctx, configLoader, configOptions, cmd.ImageSelector)
	if err != nil {
		return err
	}

	// Build options
	options := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, imageSelector, cmd.Namespace, cmd.Pod).
		WithPick(cmd.Pick).
		WithWait(cmd.Wait).
		WithContainerFilter(selector.FilterTerminatingContainers)
	if cmd.Wait {
		options = options.WithWaitingStrategy(targetselector.NewUntilNotWaitingStrategy(time.Second * 2))
	}

	// Start terminal
	err = logs.StartLogsWithWriter(ctx, targetselector.NewTargetSelector(options), cmd.Follow, int64(cmd.LastAmountOfLines), os.Stdout)
	if err != nil {
		return err
	}

	return nil
}

func getImageSelector(ctx devspacecontext.Context, configLoader loader.ConfigLoader, configOptions *loader.ConfigOptions, imageSelector string) ([]string, error) {
	var imageSelectors []string
	if imageSelector != "" {
		var (
			err          error
			config       config2.Config
			dependencies []types.Dependency
		)
		if !configLoader.Exists() {
			config = config2.Ensure(nil)
		} else {
			config, err = configLoader.Load(ctx.Context(), ctx.KubeClient(), configOptions, ctx.Log())
			if err != nil {
				return nil, err
			}

			ctx = ctx.WithConfig(config)
			dependencies, err = dependency.NewManager(ctx, configOptions).ResolveAll(ctx, dependency.ResolveOptions{})
			if err != nil {
				ctx.Log().Warnf("Error resolving dependencies: %v", err)
			}
		}

		resolved, err := runtimevar.NewRuntimeResolver(".", true).FillRuntimeVariablesAsImageSelector(ctx.Context(), imageSelector, config, dependencies)
		if err != nil {
			return nil, err
		}

		imageSelectors = append(imageSelectors, resolved.Image)
	}

	return imageSelectors, nil
}
