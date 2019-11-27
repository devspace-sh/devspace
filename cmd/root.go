package cmd

import (
	"os"

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
	"github.com/devspace-cloud/devspace/pkg/util/log"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// NewRootCmd returns a new root command
func NewRootCmd(log log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:           "devspace",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Welcome to the DevSpace!",
		PersistentPreRun: func(cobraCmd *cobra.Command, args []string) {
			if globalFlags.Silent {
				log.SetLevel(logrus.FatalLevel)
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
	f := factory.DefaultFactory()
	rootCmd := BuildRoot(f)

	// Set version for --version flag
	rootCmd.Version = upgrade.GetVersion()

	// Report any panics
	defer cloudanalytics.ReportPanics()

	// Execute command
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
	rootCmd := NewRootCmd(f.GetLog())
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags)

	// Add sub commands
	rootCmd.AddCommand(add.NewAddCmd(globalFlags))
	rootCmd.AddCommand(cleanup.NewCleanupCmd(globalFlags))
	rootCmd.AddCommand(connect.NewConnectCmd())
	rootCmd.AddCommand(create.NewCreateCmd())
	rootCmd.AddCommand(list.NewListCmd(globalFlags))
	rootCmd.AddCommand(remove.NewRemoveCmd(globalFlags))
	rootCmd.AddCommand(reset.NewResetCmd())
	rootCmd.AddCommand(set.NewSetCmd())
	rootCmd.AddCommand(status.NewStatusCmd())
	rootCmd.AddCommand(use.NewUseCmd())
	rootCmd.AddCommand(update.NewUpdateCmd(globalFlags))

	// Add main commands
	rootCmd.AddCommand(NewInitCmd(f))
	rootCmd.AddCommand(NewDevCmd(globalFlags))
	rootCmd.AddCommand(NewBuildCmd(globalFlags))
	rootCmd.AddCommand(NewSyncCmd(globalFlags))
	rootCmd.AddCommand(NewPurgeCmd(globalFlags))
	rootCmd.AddCommand(NewUpgradeCmd())
	rootCmd.AddCommand(NewDeployCmd(f, globalFlags))
	rootCmd.AddCommand(NewEnterCmd(f, globalFlags))
	rootCmd.AddCommand(NewLoginCmd())
	rootCmd.AddCommand(NewAnalyzeCmd(globalFlags))
	rootCmd.AddCommand(NewLogsCmd(globalFlags))
	rootCmd.AddCommand(NewOpenCmd(globalFlags))
	rootCmd.AddCommand(NewUICmd(globalFlags))
	rootCmd.AddCommand(NewRunCmd(globalFlags))
	rootCmd.AddCommand(NewAttachCmd(globalFlags))

	cobra.OnInitialize(func() { initConfig(f.GetLog()) })
	return rootCmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig(log log.Logger) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Panic(err)
		}

		// Search config in home directory with name ".devspace" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".devspace")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Info("Using config file:", viper.ConfigFileUsed())
	}
}
