package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/env"
	"mvdan.cc/sh/v3/expand"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/interrupt"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/loft-util/pkg/command"
	"mvdan.cc/sh/v3/interp"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/util/factory"
	flagspkg "github.com/loft-sh/devspace/pkg/util/flags"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RunCmd holds the run cmd flags
type RunCmd struct {
	*flags.GlobalFlags

	Dependency string
	Stdout     io.Writer
	Stderr     io.Writer
}

// NewRunCmd creates a new run command
func NewRunCmd(f factory.Factory, globalFlags *flags.GlobalFlags, rawConfig *RawConfig) *cobra.Command {
	cmd := &RunCmd{
		GlobalFlags: globalFlags,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	}

	runCmd := &cobra.Command{
		Use:                "run",
		DisableFlagParsing: true,
		Short:              "Executes a predefined command",
		Long: `
#######################################################
##################### devspace run ####################
#######################################################
Executes a predefined command from the devspace.yaml

Examples:
devspace run mycommand --myarg 123
devspace run mycommand2 1 2 3
devspace --dependency my-dependency run any-command --any-command-flag
#######################################################
	`,
		Args: cobra.MinimumNArgs(1),
	}
	runCmd.RunE = func(cobraCmd *cobra.Command, _ []string) error {
		args, err := ParseArgs(runCmd, cmd.GlobalFlags, f.GetLog())
		if err != nil {
			return err
		}

		plugin.SetPluginCommand(cobraCmd, args)
		return cmd.RunRun(f, args)
	}

	if rawConfig != nil && rawConfig.Config != nil {
		for _, cmd := range rawConfig.Config.Commands {
			runCmd.AddCommand(NewSpecificRunCommand(cmd))
		}
	}
	runCmd.Flags().StringVar(&cmd.Dependency, "dependency", "", "Run a command from a specific dependency")
	return runCmd
}

// RunRun executes the functionality "devspace run"
func (cmd *RunCmd) RunRun(f factory.Factory, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("run requires at least one argument")
	}

	// check if dependency command
	commandSplitted := strings.Split(args[0], ".")
	if len(commandSplitted) > 1 {
		cmd.Dependency = strings.Join(commandSplitted[:len(commandSplitted)-1], ".")
		args[0] = commandSplitted[len(commandSplitted)-1]
	}

	// Execute plugin hook
	err := hook.ExecuteHooks(nil, nil, "run")
	if err != nil {
		return err
	}

	// Set config root
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(f.GetLog())
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// load the config
	ctx, err := cmd.LoadCommandsConfig(f, configLoader, configOptions, f.GetLog())
	if err != nil {
		return err
	}

	// check if we should execute a dependency command
	if cmd.Dependency != "" {
		config, err := configLoader.LoadWithCache(context.Background(), ctx.Config().LocalCache(), nil, configOptions, f.GetLog())
		if err != nil {
			return err
		}

		ctx = ctx.WithConfig(config)
		dependencies, err := f.NewDependencyManager(ctx, configOptions).ResolveAll(ctx, dependency.ResolveOptions{})
		if err != nil {
			return err
		}

		dep := dependency.GetDependencyByPath(dependencies, cmd.Dependency)
		if dep == nil {
			return fmt.Errorf("couldn't find dependency %s", cmd.Dependency)
		}

		ctx = ctx.AsDependency(dep)
		commandConfig, err := findCommand(ctx.Config(), args[0])
		if err != nil {
			return err
		}

		return executeCommandWithAfter(ctx.Context(), commandConfig, args[1:], ctx.Config().Variables(), ctx.WorkingDir(), cmd.Stdout, cmd.Stderr, os.Stdin, ctx.Log())
	}

	commandConfig, err := findCommand(ctx.Config(), args[0])
	if err != nil {
		return err
	}

	return executeCommandWithAfter(ctx.Context(), commandConfig, args[1:], ctx.Config().Variables(), ctx.WorkingDir(), cmd.Stdout, cmd.Stderr, os.Stdin, ctx.Log())
}

func findCommand(config config.Config, name string) (*latest.CommandConfig, error) {
	// Find command
	if config.Config().Commands == nil || config.Config().Commands[name] == nil {
		return nil, errors.Errorf("couldn't find command '%s' in devspace config", name)
	}

	return config.Config().Commands[name], nil
}

func executeCommandWithAfter(ctx context.Context, command *latest.CommandConfig, args []string, variables map[string]interface{}, dir string, stdout io.Writer, stderr io.Writer, stdin io.Reader, log log.Logger) error {
	originalErr := interrupt.Global.Run(func() error {
		return ExecuteCommand(ctx, command, variables, args, dir, stdout, stderr, stdin)
	}, func() {
		if command.After != "" {
			vars := variables
			vars["COMMAND_INTERRUPT"] = "true"
			err := executeShellCommand(ctx, command.After, vars, args, dir, stdout, stderr, stdin)
			if err != nil {
				log.Errorf("error executing after command: %v", err)
			}
		}
	})
	if command.After != "" {
		vars := variables
		if originalErr != nil {
			vars["COMMAND_ERROR"] = originalErr.Error()
		}
		err := executeShellCommand(ctx, command.After, vars, args, dir, stdout, stderr, stdin)
		if err != nil {
			return errors.Wrap(err, "error executing after command")
		}
	}

	return originalErr
}

func ParseArgs(cobraCmd *cobra.Command, globalFlags *flags.GlobalFlags, log log.Logger) ([]string, error) {
	index := -1
	for i, v := range os.Args {
		if v == cobraCmd.Use {
			index = i + 1
			break
		}
	}
	if index == -1 {
		return nil, fmt.Errorf("error parsing command: couldn't find %s in command: %v", cobraCmd.Use, os.Args)
	}

	// check if is help command
	osArgs := os.Args[:index]
	if len(os.Args) == index+1 && (os.Args[index] == "-h" || os.Args[index] == "--help") {
		return nil, cobraCmd.Help()
	}

	// enable flag parsing
	cobraCmd.DisableFlagParsing = false

	// apply extra flags
	_, err := flagspkg.ApplyExtraFlags(cobraCmd, osArgs, true)
	if err != nil {
		return nil, err
	}

	if globalFlags.Silent {
		log.SetLevel(logrus.FatalLevel)
	} else if globalFlags.Debug {
		log.SetLevel(logrus.DebugLevel)
	}

	args := os.Args[index:]
	return args, nil
}

// LoadCommandsConfig loads the commands config
func (cmd *RunCmd) LoadCommandsConfig(f factory.Factory, configLoader loader.ConfigLoader, configOptions *loader.ConfigOptions, log log.Logger) (devspacecontext.Context, error) {
	// load generated
	localCache, err := configLoader.LoadLocalCache()
	if err != nil {
		return nil, err
	}

	// try to load client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		log.Debugf("Unable to create new kubectl client: %v", err)
		client = nil
	}

	// verify client connectivity / authn / authz
	if client != nil {
		// If the current kube context or namespace is different than old,
		// show warnings and reset kube client if necessary
		client, err = kubectl.CheckKubeContext(client, localCache, false, false, false, log)
		if err != nil {
			log.Debugf("Unable to verify kube context %v", err)
			client = nil
		}
	}

	// Parse commands
	commandsInterface, err := configLoader.LoadWithParser(context.Background(), localCache, client, loader.NewCommandsParser(), configOptions, log)
	if err != nil {
		return nil, err
	}

	// create context
	return devspacecontext.NewContext(context.Background(), commandsInterface.Variables(), log).
		WithKubeClient(client).
		WithConfig(commandsInterface), nil
}

func executeShellCommand(ctx context.Context, shellCommand string, variables map[string]interface{}, args []string, dir string, stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	extraEnv := map[string]string{}
	for k, v := range variables {
		extraEnv[k] = fmt.Sprintf("%v", v)
	}

	// execute the command in a shell
	err := engine.ExecuteSimpleShellCommand(ctx, dir, env.NewVariableEnvProvider(expand.ListEnviron(os.Environ()...), extraEnv), stdout, stderr, stdin, shellCommand, args...)
	if err != nil {
		if status, ok := interp.IsExitStatus(err); ok {
			return &exit.ReturnCodeError{
				ExitCode: int(status),
			}
		}

		return errors.Wrap(err, "execute command")
	}

	return nil
}

// ExecuteCommand executes a command from the config
func ExecuteCommand(ctx context.Context, cmd *latest.CommandConfig, variables map[string]interface{}, args []string, dir string, stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	shellCommand := strings.TrimSpace(cmd.Command)
	shellArgs := cmd.Args
	appendArgs := cmd.AppendArgs

	extraEnv := map[string]string{}
	for k, v := range variables {
		extraEnv[k] = fmt.Sprintf("%v", v)
	}
	if shellArgs == nil {
		if appendArgs {
			// Append args to shell command
			for _, arg := range args {
				arg = strings.ReplaceAll(arg, "'", "'\"'\"'")

				shellCommand += " '" + arg + "'"
			}
		}

		// execute the command in a shell
		err := engine.ExecuteSimpleShellCommand(ctx, dir, env.NewVariableEnvProvider(expand.ListEnviron(os.Environ()...), extraEnv), stdout, stderr, stdin, shellCommand, args...)
		if err != nil {
			if status, ok := interp.IsExitStatus(err); ok {
				return &exit.ReturnCodeError{
					ExitCode: int(status),
				}
			}

			return errors.Wrap(err, "execute command")
		}

		return nil
	}

	shellArgs = append(shellArgs, args...)
	return command.Command(ctx, dir, env.NewVariableEnvProvider(expand.ListEnviron(os.Environ()...), extraEnv), stdout, stderr, stdin, shellCommand, shellArgs...)
}

// RunCommandCmd holds the cmd flags of a run command
type RunCommandCmd struct {
	*flags.GlobalFlags

	Command   *latest.CommandConfig
	Variables map[string]interface{}

	Stdout io.Writer
	Stderr io.Writer
}

// NewSpecificRunCommand creates a new run command
func NewSpecificRunCommand(command *latest.CommandConfig) *cobra.Command {
	description := command.Description
	longDescription := command.Description
	if description == "" {
		description = "Runs command: " + command.Name
		longDescription = description
	}
	if len(description) > 64 {
		if len(description) > 64 {
			description = description[:61] + "..."
		}
	}

	runCmd := &cobra.Command{
		Use:   command.Name,
		Short: description,
		Long:  longDescription,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cobraCmd *cobra.Command, originalArgs []string) error {
			return cobraCmd.Parent().RunE(cobraCmd, originalArgs)
		},
	}
	runCmd.DisableFlagParsing = true
	return runCmd
}
