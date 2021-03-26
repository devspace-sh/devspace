package cmd

import (
	"flag"
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
	"github.com/loft-sh/devspace/cmd/status"
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
	"io/ioutil"
	"k8s.io/klog"
	"os"
	"strings"
	"time"
)

// NewRootCmd returns a new root command
func NewRootCmd(f factory.Factory, plugins []plugin.Metadata) *cobra.Command {
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
			}

			// parse the .env file
			err := godotenv.Load()
			if err != nil && os.IsNotExist(err) == false {
				log.Warnf("Error loading .env: %v", err)
			}

			// apply extra flags
			if cobraCmd.DisableFlagParsing == false {
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
			err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "root", "", "", nil)
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
	rootCmd := BuildRoot(f)

	// set version for --version flag
	rootCmd.Version = upgrade.GetVersion()

	// execute command
	err := rootCmd.Execute()
	if err != nil {
		// Check if return code error
		retCode, ok := errors.Cause(err).(*exit.ReturnCodeError)
		if ok {
			os.Exit(retCode.ExitCode)
		}

		if globalFlags.Debug {
			f.GetLog().Fatalf("%+v", err)
		} else {
			f.GetLog().Fatal(err)
		}
	}
}

// BuildRoot creates a new root command from the
func BuildRoot(f factory.Factory) *cobra.Command {
	// list plugins
	plugins, err := f.NewPluginManager(f.GetLog()).List()
	if err != nil {
		f.GetLog().Fatal(err)
	}

	// build the root cmd
	rootCmd := NewRootCmd(f, plugins)
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags)

	// Add sub commands
	rootCmd.AddCommand(add.NewAddCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(cleanup.NewCleanupCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(list.NewListCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(remove.NewRemoveCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(reset.NewResetCmd(f, plugins))
	rootCmd.AddCommand(set.NewSetCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(status.NewStatusCmd(f, plugins))
	rootCmd.AddCommand(use.NewUseCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(update.NewUpdateCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(save.NewSaveCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(restore.NewRestoreCmd(f, globalFlags, plugins))

	// Add main commands
	rootCmd.AddCommand(NewInitCmd(f, plugins))
	rootCmd.AddCommand(NewRestartCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewDevCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewBuildCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewSyncCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewRenderCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewPurgeCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewUpgradeCmd(plugins))
	rootCmd.AddCommand(NewDeployCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewEnterCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewAnalyzeCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewLogsCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewOpenCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewUICmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewRunCmd(f, globalFlags))
	rootCmd.AddCommand(NewAttachCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewPrintCmd(f, globalFlags, plugins))

	// Add plugin commands
	plugin.AddPluginCommands(rootCmd, plugins, "")
	variable.AddPredefinedVars(plugins)
	return rootCmd
}

func disableKlog() {
	flagSet := &flag.FlagSet{}
	klog.InitFlags(flagSet)
	flagSet.Set("logtostderr", "false")
	klog.SetOutput(ioutil.Discard)
}
