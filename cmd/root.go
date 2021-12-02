package cmd

import (
	"flag"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/util/interrupt"

	"github.com/joho/godotenv"
	"github.com/loft-sh/devspace/cmd/add"
	"github.com/loft-sh/devspace/cmd/cleanup"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/cmd/list"
	"github.com/loft-sh/devspace/cmd/remove"
	"github.com/loft-sh/devspace/cmd/reset"
	"github.com/loft-sh/devspace/cmd/restore"
	"github.com/loft-sh/devspace/cmd/save"
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

			// parse the .env file
			err := godotenv.Load()
			if err != nil && !os.IsNotExist(err) {
				log.Warnf("Error loading .env: %v", err)
			}

			// apply extra flags
			if !cobraCmd.DisableFlagParsing {
				extraFlags, err := flagspkg.ApplyExtraFlags(cobraCmd, os.Args, false)
				if err != nil {
					log.Warnf("Error applying extra flags: %v", err)
				} else if len(extraFlags) > 0 {
					log.Debugf("Applying extra flags from environment: %s", strings.Join(extraFlags, " "))
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
	
		devspace init`,
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
	pluginErr := hook.ExecuteHooks(nil, nil, nil, nil, nil, "root", "root.beforeExecute", "command:before:execute")
	if pluginErr != nil {
		f.GetLog().Fatalf("%+v", pluginErr)
	}

	// execute command
	err := rootCmd.Execute()

	// after hooks
	pluginErr = hook.ExecuteHooks(nil, nil, nil, map[string]interface{}{"error": err}, nil, "root.afterExecute", "command:after:execute")
	if err != nil {
		// Check if return code error
		retCode, ok := errors.Cause(err).(*exit.ReturnCodeError)
		if ok {
			os.Exit(retCode.ExitCode)
		}

		// error hooks
		pluginErr := hook.ExecuteHooks(nil, nil, nil, map[string]interface{}{"error": err}, nil, "root.errorExecution", "command:error")
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

	// build the root cmd
	rootCmd := NewRootCmd(f)
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags)

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

{{- if (and .HasAvailableSubCommands) -}}
{{- range .Commands -}}
{{- if (and .HasSubCommands (eq .Name "run"))}}

Additional run commands:
{{- range .Commands}}
  {{rpad (printf "'%s'" .CommandPath) .CommandPathPadding}} {{.Short}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
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
	rootCmd.AddCommand(save.NewSaveCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(restore.NewRestoreCmd(f, globalFlags, plugins))

	// Add main commands
	rootCmd.AddCommand(NewInitCmd(f))
	rootCmd.AddCommand(NewRestartCmd(f, globalFlags))
	rootCmd.AddCommand(NewDevCmd(f, globalFlags))
	rootCmd.AddCommand(NewBuildCmd(f, globalFlags))
	rootCmd.AddCommand(NewSyncCmd(f, globalFlags))
	rootCmd.AddCommand(NewRenderCmd(f, globalFlags))
	rootCmd.AddCommand(NewPurgeCmd(f, globalFlags))
	rootCmd.AddCommand(NewUpgradeCmd())
	rootCmd.AddCommand(NewDeployCmd(f, globalFlags))
	rootCmd.AddCommand(NewEnterCmd(f, globalFlags))
	rootCmd.AddCommand(NewAnalyzeCmd(f, globalFlags))
	rootCmd.AddCommand(NewLogsCmd(f, globalFlags))
	rootCmd.AddCommand(NewOpenCmd(f, globalFlags))
	rootCmd.AddCommand(NewUICmd(f, globalFlags))
	rootCmd.AddCommand(NewRunCmd(f, globalFlags))
	rootCmd.AddCommand(NewAttachCmd(f, globalFlags))
	rootCmd.AddCommand(NewPrintCmd(f, globalFlags))
	rootCmd.AddCommand(NewCompletionCmd())

	// Add plugin commands
	plugin.AddPluginCommands(rootCmd, plugins, "")
	variable.AddPredefinedVars(plugins)
	return rootCmd
}

func disableKlog() {
	flagSet := &flag.FlagSet{}
	klog.InitFlags(flagSet)
	_ = flagSet.Set("logtostderr", "false")
	klog.SetOutput(ioutil.Discard)

	flagSet = &flag.FlagSet{}
	klogv2.InitFlags(flagSet)
	_ = flagSet.Set("logtostderr", "false")
	klogv2.SetOutput(ioutil.Discard)
}
