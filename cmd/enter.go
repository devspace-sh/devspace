package cmd

import (
	"context"
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/terminal"
	"io"
	"strings"
	"time"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// EnterCmd is a struct that defines a command call for "enter"
type EnterCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	ImageSelector string
	Container     string
	Pod           string
	Pick          bool
	TTY           bool
	Wait          bool
	Reconnect     bool
	Screen        bool
	ScreenSession string

	WorkingDirectory string

	// used for testing
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

// NewEnterCmd creates a new enter command
func NewEnterCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &EnterCmd{GlobalFlags: globalFlags}

	enterCmd := &cobra.Command{
		Use:   "enter",
		Short: "Open a shell to a container",
		Long: `
#######################################################
################## devspace enter #####################
#######################################################
Execute a command or start a new terminal in your 
devspace:

devspace enter
devspace enter --pick # Select pod to enter
devspace enter bash
devspace enter -c my-container
devspace enter bash -n my-namespace
devspace enter bash -l release=test
devspace enter bash --image-selector nginx:latest
devspace enter bash --image-selector "${runtime.images.app.image}:${runtime.images.app.tag}"
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, args)
		},
	}

	enterCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	enterCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	enterCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	enterCmd.Flags().StringVar(&cmd.ImageSelector, "image-selector", "", "The image to search a pod for (e.g. nginx, nginx:latest, ${runtime.images.app}, nginx:${runtime.images.app.tag})")
	enterCmd.Flags().StringVar(&cmd.WorkingDirectory, "workdir", "", "The working directory where to open the terminal or execute the command")

	enterCmd.Flags().BoolVar(&cmd.TTY, "tty", true, "If to use a tty to start the command")
	enterCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod / container if multiple are found")
	enterCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "Wait for the pod(s) to start if they are not running")
	enterCmd.Flags().BoolVar(&cmd.Reconnect, "reconnect", false, "Will reconnect the terminal if an unexpected return code is encountered")
	enterCmd.Flags().BoolVar(&cmd.Screen, "screen", false, "Use a screen session to connect")
	enterCmd.Flags().StringVar(&cmd.ScreenSession, "screen-session", "enter", "The screen session to create or connect to")

	return enterCmd
}

// Run executes the command logic
func (cmd *EnterCmd) Run(f factory.Factory, args []string) error {
	// Set config root
	logger := f.GetLog()
	configOptions := cmd.ToConfigOptions()
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
	if configExists {
		localCache, err := configLoader.LoadLocalCache()
		if err != nil {
			return err
		}

		// If the current kube context or namespace is different from old,
		// show warnings and reset kube client if necessary
		client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, logger)
		if err != nil {
			return err
		}
	}

	// create the context
	ctx := devspacecontext.NewContext(context.Background(), nil, logger).WithKubeClient(client)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "enter")
	if err != nil {
		return err
	}

	// get image selector if specified
	imageSelector, err := getImageSelector(ctx, configLoader, configOptions, cmd.ImageSelector)
	if err != nil {
		return err
	}

	// Build params
	selectorOptions := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, imageSelector, cmd.Namespace, cmd.Pod).
		WithPick(cmd.Pick).
		WithWait(cmd.Wait).
		WithQuestion("Which pod do you want to open the terminal for?")
	if cmd.Wait {
		selectorOptions = selectorOptions.WithContainerFilter(selector.FilterTerminatingContainers)
		selectorOptions = selectorOptions.WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Second))
	}

	// build command
	command := []string{"sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"}
	if len(args) > 0 {
		command = args
	}
	if cmd.WorkingDirectory != "" {
		command = []string{"sh", "-c", fmt.Sprintf("cd %s; %s", cmd.WorkingDirectory, strings.Join(command, " "))}
	}

	// Start terminal
	stdout, stderr, stdin := defaultStdStreams(cmd.Stdout, cmd.Stderr, cmd.Stdin)
	exitCode, err := terminal.StartTerminalFromCMD(ctx, targetselector.NewTargetSelector(selectorOptions), command, cmd.Wait, cmd.Reconnect, cmd.TTY, cmd.Screen, cmd.ScreenSession, stdout, stderr, stdin)
	if err != nil {
		return err
	} else if exitCode != 0 {
		return &exit.ReturnCodeError{
			ExitCode: exitCode,
		}
	}

	return nil
}
