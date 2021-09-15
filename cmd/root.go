package cmd

import (
	"flag"
	"io/ioutil"
	"os"
	"strings"
	"time"

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
					log.Infof("Applying extra flags from environment: %s", strings.Join(extraFlags, " "))
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

			// call root plugin hook
			err = plugin.ExecutePluginHook("root")
			if err != nil {
				return err
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
	// disable klog
	disableKlog()

	// create a new factory
	f := factory.DefaultFactory()

	// build the root command
	rootCmd := BuildRoot(f, false)

	// set version for --version flag
	rootCmd.Version = upgrade.GetVersion()

	// call root plugin hook
	err := plugin.ExecutePluginHook("root.beforeExecute")
	if err != nil {
		f.GetLog().Fatal(err)
	}

	// execute command
	err = rootCmd.Execute()
	if err != nil {
		// Check if return code error
		retCode, ok := errors.Cause(err).(*exit.ReturnCodeError)
		if ok {
			os.Exit(retCode.ExitCode)
		}

		// call root plugin hook
		pluginErr := plugin.ExecutePluginHookWithContext("root.errorExecution", map[string]interface{}{
			"error": err,
		})
		if pluginErr != nil {
			f.GetLog().Fatal(pluginErr)
		}

		if globalFlags.Debug {
			f.GetLog().Fatalf("%+v", err)
		} else {
			f.GetLog().Fatal(err)
		}
	}

	// call root plugin hook
	err = plugin.ExecutePluginHook("root.afterExecute")
	if err != nil {
		f.GetLog().Fatal(err)
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
}
