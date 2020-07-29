package cmd

import (
	"github.com/devspace-cloud/devspace/cmd/add"
	"github.com/devspace-cloud/devspace/cmd/cleanup"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/cmd/list"
	"github.com/devspace-cloud/devspace/cmd/remove"
	"github.com/devspace-cloud/devspace/cmd/reset"
	"github.com/devspace-cloud/devspace/cmd/set"
	"github.com/devspace-cloud/devspace/cmd/status"
	"github.com/devspace-cloud/devspace/cmd/update"
	"github.com/devspace-cloud/devspace/cmd/use"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"
	"github.com/devspace-cloud/devspace/pkg/util/analytics/cloudanalytics"
	"github.com/devspace-cloud/devspace/pkg/util/exit"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	flagspkg "github.com/devspace-cloud/devspace/pkg/util/flags"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// NewRootCmd returns a new root command
func NewRootCmd(f factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:           "devspace",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Welcome to the DevSpace!",
		PersistentPreRun: func(cobraCmd *cobra.Command, args []string) {
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
			}

			// Get version of current binary
			latestVersion := upgrade.NewerVersionAvailable()
			if latestVersion != "" {
				log.Warnf("There is a newer version of DevSpace: v%s. Run `devspace upgrade` to upgrade to the newest version.\n", latestVersion)
			}
		},
		Long: `DevSpace accelerates developing, deploying and debugging applications with Docker and Kubernetes. Get started by running the init command in one of your projects:
	
		devspace init`,
	}
}

var globalFlags *flags.GlobalFlags

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// report any panics
	defer cloudanalytics.ReportPanics()

	// create a new factory
	f := factory.DefaultFactory()

	// build the root command
	rootCmd := BuildRoot(f)

	// set version for --version flag
	rootCmd.Version = upgrade.GetVersion()

	// execute command
	err := rootCmd.Execute()
	cloudanalytics.SendCommandEvent(err)
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
	rootCmd := NewRootCmd(f)
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags)

	// list plugins
	plugins, err := f.NewPluginManager(f.GetLog()).List()
	if err != nil {
		f.GetLog().Fatal(err)
	}

	// Add sub commands
	rootCmd.AddCommand(add.NewAddCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(cleanup.NewCleanupCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(list.NewListCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(remove.NewRemoveCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(reset.NewResetCmd(f, plugins))
	rootCmd.AddCommand(set.NewSetCmd(f, plugins))
	rootCmd.AddCommand(status.NewStatusCmd(f, plugins))
	rootCmd.AddCommand(use.NewUseCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(update.NewUpdateCmd(f, globalFlags, plugins))

	// Add main commands
	rootCmd.AddCommand(NewInitCmd(f))
	rootCmd.AddCommand(NewDevCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewBuildCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewSyncCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewRenderCmd(f, globalFlags))
	rootCmd.AddCommand(NewPurgeCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewUpgradeCmd())
	rootCmd.AddCommand(NewDeployCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewEnterCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewAnalyzeCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewLogsCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewOpenCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewUICmd(f, globalFlags))
	rootCmd.AddCommand(NewRunCmd(f, globalFlags))
	rootCmd.AddCommand(NewAttachCmd(f, globalFlags, plugins))
	rootCmd.AddCommand(NewPrintCmd(f, globalFlags))

	// Add plugin commands
	plugin.AddPluginCommands(rootCmd, plugins, "")
	loader.AddPredefinedVars(plugins)
	return rootCmd
}
