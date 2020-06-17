package cmd

import (
	"github.com/devspace-cloud/devspace/cmd/add"
	"github.com/devspace-cloud/devspace/cmd/cleanup"
	"github.com/devspace-cloud/devspace/cmd/connect"
	"github.com/devspace-cloud/devspace/cmd/create"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/cmd/list"
	"github.com/devspace-cloud/devspace/cmd/remove"
	"github.com/devspace-cloud/devspace/cmd/reset"
	"github.com/devspace-cloud/devspace/cmd/set"
	"github.com/devspace-cloud/devspace/cmd/status"
	"github.com/devspace-cloud/devspace/cmd/update"
	"github.com/devspace-cloud/devspace/cmd/use"
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"
	"github.com/devspace-cloud/devspace/pkg/util/analytics/cloudanalytics"
	"github.com/devspace-cloud/devspace/pkg/util/exit"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	flagspkg "github.com/devspace-cloud/devspace/pkg/util/flags"
	"github.com/joho/godotenv"
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
				extraFlags, err := flagspkg.ApplyExtraFlags(cobraCmd, os.Args)
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
		retCode, ok := err.(*exit.ReturnCodeError)
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

	// Add sub commands
	rootCmd.AddCommand(add.NewAddCmd(f, globalFlags))
	rootCmd.AddCommand(cleanup.NewCleanupCmd(f, globalFlags))
	rootCmd.AddCommand(connect.NewConnectCmd(f))
	rootCmd.AddCommand(create.NewCreateCmd(f))
	rootCmd.AddCommand(list.NewListCmd(f, globalFlags))
	rootCmd.AddCommand(remove.NewRemoveCmd(f, globalFlags))
	rootCmd.AddCommand(reset.NewResetCmd(f))
	rootCmd.AddCommand(set.NewSetCmd(f))
	rootCmd.AddCommand(status.NewStatusCmd(f))
	rootCmd.AddCommand(use.NewUseCmd(f, globalFlags))
	rootCmd.AddCommand(update.NewUpdateCmd(f, globalFlags))

	// Add main commands
	rootCmd.AddCommand(NewInitCmd(f))
	rootCmd.AddCommand(NewDevCmd(f, globalFlags))
	rootCmd.AddCommand(NewBuildCmd(f, globalFlags))
	rootCmd.AddCommand(NewSyncCmd(f, globalFlags))
	rootCmd.AddCommand(NewRenderCmd(f, globalFlags))
	rootCmd.AddCommand(NewPurgeCmd(f, globalFlags))
	rootCmd.AddCommand(NewUpgradeCmd())
	rootCmd.AddCommand(NewDeployCmd(f, globalFlags))
	rootCmd.AddCommand(NewEnterCmd(f, globalFlags))
	rootCmd.AddCommand(NewLoginCmd(f))
	rootCmd.AddCommand(NewAnalyzeCmd(f, globalFlags))
	rootCmd.AddCommand(NewLogsCmd(f, globalFlags))
	rootCmd.AddCommand(NewOpenCmd(f, globalFlags))
	rootCmd.AddCommand(NewUICmd(f, globalFlags))
	rootCmd.AddCommand(NewRunCmd(f, globalFlags))
	rootCmd.AddCommand(NewAttachCmd(f, globalFlags))
	rootCmd.AddCommand(NewPrintCmd(f, globalFlags))

	return rootCmd
}
