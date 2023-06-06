package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/kill"
	"github.com/mgutz/ansi"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/expression"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/env"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"

	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/util/interrupt"

	"github.com/joho/godotenv"
	"github.com/loft-sh/devspace/cmd/add"
	"github.com/loft-sh/devspace/cmd/cleanup"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/cmd/list"
	"github.com/loft-sh/devspace/cmd/remove"
	"github.com/loft-sh/devspace/cmd/reset"
	"github.com/loft-sh/devspace/cmd/set"
	"github.com/loft-sh/devspace/cmd/update"
	"github.com/loft-sh/devspace/cmd/use"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/factory"
	flagspkg "github.com/loft-sh/devspace/pkg/util/flags"
	"github.com/loft-sh/devspace/pkg/util/idle"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/klog"
	klogv2 "k8s.io/klog/v2"
)

// NewRootCmd returns a new root command
func NewRootCmd(f factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:           "devspace",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Welcome to the DevSpace!",
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			// don't do anything if it is a plugin command
			if cobraCmd.Annotations != nil && cobraCmd.Annotations[plugin.PluginCommandAnnotation] == "true" {
				return nil
			}

			log := f.GetLog()
			if globalFlags.Silent {
				log.SetLevel(logrus.FatalLevel)
			} else if globalFlags.Debug {
				log.SetLevel(logrus.DebugLevel)
			}

			ansi.DisableColors(globalFlags.NoColors)

			if globalFlags.KubeConfig != "" {
				err := os.Setenv("KUBECONFIG", globalFlags.KubeConfig)
				if err != nil {
					log.Errorf("Unable to set KUBECONFIG variable: %v", err)
				}
			}

			// parse the .env file
			envFile := env.GlobalGetEnv("DEVSPACE_ENV_FILE")
			if envFile != "" {
				err := godotenv.Load(envFile)
				if err != nil && !os.IsNotExist(err) {
					log.Warnf("Error loading .env: %v", err)
				}
			}

			// apply extra flags
			if !cobraCmd.DisableFlagParsing {
				extraFlags, err := flagspkg.ApplyExtraFlags(cobraCmd, os.Args, false)
				if err != nil {
					log.Warnf("Error applying extra flags: %v", err)
				} else if len(extraFlags) > 0 {
					log.Debugf("Applying extra flags from environment: %s", strings.Join(extraFlags, " "))
				}

				if globalFlags.Silent {
					log.SetLevel(logrus.FatalLevel)
				} else if globalFlags.Debug {
					log.SetLevel(logrus.DebugLevel)
				}

				// call inactivity timeout
				if globalFlags.InactivityTimeout > 0 {
					m, err := idle.NewIdleMonitor()
					if err != nil {
						log.Warnf("Error creating inactivity monitor: %v", err)
					} else if m != nil {
						m.Start(time.Duration(globalFlags.InactivityTimeout)*time.Minute, log)
					}
				}
			}

			return nil
		},
		Long: `DevSpace accelerates developing, deploying and debugging applications with Docker and Kubernetes. Get started by running the init command in one of your projects:
	
		devspace init
		# Develop an existing application
		devspace dev
		DEVSPACE_CONFIG=other-config.yaml devspace dev`,
	}
}

var globalFlags *flags.GlobalFlags

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	interrupt.Global.Start()

	// disable klog
	disableKlog()

	// create a new factory
	f := factory.DefaultFactory()

	// build the root command
	rootCmd := BuildRoot(f, false)

	// set version for --version flag
	rootCmd.Version = upgrade.GetVersion()

	// before hooks
	pluginErr := hook.ExecuteHooks(nil, nil, "root", "root.beforeExecute", "command:before:execute")
	if pluginErr != nil {
		f.GetLog().Fatalf("%+v", pluginErr)
	}

	// execute command
	err := rootCmd.Execute()

	// after hooks
	pluginErr = hook.ExecuteHooks(nil, map[string]interface{}{"error": err}, "root.afterExecute", "command:after:execute")
	if err != nil {
		// Check if return code error
		retCode, ok := errors.Cause(err).(*exit.ReturnCodeError)
		if ok {
			os.Exit(retCode.ExitCode)
		}

		// error hooks
		pluginErr := hook.ExecuteHooks(nil, map[string]interface{}{"error": err}, "root.errorExecution", "command:error")
		if pluginErr != nil {
			f.GetLog().Fatalf("%+v", pluginErr)
		}

		if globalFlags.Debug {
			f.GetLog().Fatalf("%+v", err)
		} else {
			f.GetLog().Fatal(err)
		}
	} else if pluginErr != nil {
		f.GetLog().Fatalf("%+v", pluginErr)
	}
}

// BuildRoot creates a new root command from the
func BuildRoot(f factory.Factory, excludePlugins bool) *cobra.Command {
	// list plugins
	var (
		plugins []plugin.Metadata
		err     error
	)
	if !excludePlugins {
		plugins, err = f.NewPluginManager(f.GetLog()).List()
		if err != nil {
			f.GetLog().Fatal(err)
		}

		plugin.SetPlugins(plugins)
	}

	// try to parse the raw config
	var rawConfig *RawConfig

	// This check is necessary to avoid process loops where a variable inside
	// the devspace.yaml would execute another devspace command which would again
	// load the config and execute DevSpace config parsing etc.
	if os.Getenv(expression.DevSpaceSkipPreloadEnv) == "" {
		rawConfig, err = parseConfig(f)
		if err != nil {
			f.GetLog().Debugf("error parsing raw config: %v", err)
		} else {
			env.GlobalGetEnv = rawConfig.GetEnv
		}
	}

	// build the root cmd
	rootCmd := NewRootCmd(f)
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags)
	kill.SetStopFunction(func(message string) {
		if message == "" {
			os.Exit(1)
		} else {
			f.GetLog().Fatal(message)
		}
	})

	rootCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}
  
Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}
  
Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}
  
Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}
  
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}
  
Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{- if not (eq .Name "run")}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
{{- end -}}

{{- if (and .HasAvailableSubCommands) -}}
{{- range .Commands -}}
{{- if (and .HasSubCommands (eq .Name "run"))}}

Additional run commands:
{{- range .Commands}}
  {{rpad (printf "'%s'" .CommandPath) .CommandPathPadding}} {{.Short}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{end}}
`)

	// Add sub commands
	rootCmd.AddCommand(add.NewAddCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(cleanup.NewCleanupCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(list.NewListCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(remove.NewRemoveCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(reset.NewResetCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(set.NewSetCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(use.NewUseCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(update.NewUpdateCmd(f, globalFlags, plugins))

	// Add main commands
	rootCmd.AddCommand(NewInitCmd(f))
	rootCmd.AddCommand(NewRestartCmd(f, globalFlags))
	rootCmd.AddCommand(NewSyncCmd(f, globalFlags))
	rootCmd.AddCommand(NewRenderCmd(f, globalFlags, rawConfig))
	rootCmd.AddCommand(NewUpgradeCmd())
	rootCmd.AddCommand(NewEnterCmd(f, globalFlags))
	rootCmd.AddCommand(NewAnalyzeCmd(f, globalFlags))
	rootCmd.AddCommand(NewLogsCmd(f, globalFlags))
	rootCmd.AddCommand(NewOpenCmd(f, globalFlags))
	rootCmd.AddCommand(NewUICmd(f, globalFlags))
	rootCmd.AddCommand(NewRunCmd(f, globalFlags, rawConfig))
	rootCmd.AddCommand(NewAttachCmd(f, globalFlags))
	rootCmd.AddCommand(NewPrintCmd(f, globalFlags))
	rootCmd.AddCommand(NewRunPipelineCmd(f, globalFlags, rawConfig))
	rootCmd.AddCommand(NewCompletionCmd())
	rootCmd.AddCommand(NewVersionCmd())

	// check overwrite commands
	rootCmd.AddCommand(NewDevCmd(f, globalFlags, rawConfig))
	rootCmd.AddCommand(NewDeployCmd(f, globalFlags, rawConfig))
	rootCmd.AddCommand(NewBuildCmd(f, globalFlags, rawConfig))
	rootCmd.AddCommand(NewPurgeCmd(f, globalFlags, rawConfig))

	// Add plugin commands
	if rawConfig != nil && rawConfig.OriginalRawConfig != nil {
		plugin.AddDevspaceVarsToPluginEnv(rawConfig.OriginalRawConfig["vars"])
	}
	plugin.AddPluginCommands(rootCmd, plugins, "")
	variable.AddPredefinedVars(plugins)
	return rootCmd
}

func disableKlog() {
	flagSet := &flag.FlagSet{}
	klog.InitFlags(flagSet)
	_ = flagSet.Set("logtostderr", "false")
	klog.SetOutput(io.Discard)

	flagSet = &flag.FlagSet{}
	klogv2.InitFlags(flagSet)
	_ = flagSet.Set("logtostderr", "false")
	klogv2.SetOutput(io.Discard)
}

func parseConfig(f factory.Factory) (*RawConfig, error) {
	// get current working dir
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// set working dir back to original
	defer func() { _ = os.Chdir(cwd) }()

	// Set config root
	configLoader, err := f.NewConfigLoader("")
	if err != nil {
		return nil, err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log.Discard)
	if err != nil {
		return nil, err
	} else if !configExists {
		return nil, errors.New(message.ConfigNotFound)
	}

	// Parse commands
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	r := &RawConfig{
		resolved: map[string]string{},
	}
	_, err = configLoader.LoadWithParser(timeoutCtx, nil, nil, r, &loader.ConfigOptions{
		Dry: true,
	}, log.Discard)
	if r.Resolver != nil {
		return r, nil
	}

	return nil, err
}

type RawConfig struct {
	Ctx               context.Context
	OriginalRawConfig map[string]interface{}
	RawConfig         map[string]interface{}
	Resolver          variable.Resolver

	Config *latest.Config

	resolvedMutex sync.Mutex
	resolved      map[string]string
}

func (r *RawConfig) Parse(
	ctx context.Context,
	originalRawConfig map[string]interface{},
	rawConfig map[string]interface{},
	resolver variable.Resolver,
	log log.Logger,
) (*latest.Config, map[string]interface{}, error) {
	r.Ctx = ctx
	r.OriginalRawConfig = originalRawConfig
	r.RawConfig = rawConfig
	r.Resolver = resolver

	// try parsing commands
	latestConfig, beforeConversion, err := loader.NewCommandsPipelinesParser().Parse(ctx, originalRawConfig, rawConfig, resolver, log)
	r.Config = latestConfig
	return latestConfig, beforeConversion, err
}

func (r *RawConfig) GetEnv(name string) string {
	// try to get from environment
	value := os.Getenv(name)
	if value != "" {
		return value
	}

	// try to find devspace variable
	if r.Resolver != nil {
		r.resolvedMutex.Lock()
		defer r.resolvedMutex.Unlock()

		// cache which ones were tried
		value, ok := r.resolved[name]
		if ok {
			return value
		}

		varName := "${" + name + "}"
		out, err := r.Resolver.FillVariables(r.Ctx, varName, true)
		if err == nil {
			value := fmt.Sprintf("%v", out)
			if value != varName && value != "" {
				r.resolved[name] = value
				return value
			}
		}

		r.resolved[name] = ""
	}

	return ""
}
