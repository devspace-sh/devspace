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
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/joho/godotenv"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var cfgFile string

// NewRootCmd returns a new root command
func NewRootCmd(f factory.Factory) *cobra.Command {
	log := f.GetLog()
	// we delay the output because we don't want to print things when we are silenced
	warnings := []string{}

	// parse the .env file
	err := godotenv.Load()
	if err != nil && os.IsNotExist(err) == false {
		warnings = append(warnings, "Error loading .env: " + err.Error())
	}

	// parse the environment flags
	extraFlags, err := parseEnvironmentFlags(f)
	if err != nil {
		warnings = append(warnings, "Error parsing environment variables: " + err.Error())
	}

	return &cobra.Command{
		Use:           "devspace",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Welcome to the DevSpace!",
		PersistentPreRun: func(cobraCmd *cobra.Command, args []string) {
			if globalFlags.Silent {
				log.SetLevel(logrus.FatalLevel)
			}

			if len(extraFlags) > 0 {
				log.Infof("Applying extra flags from environment: %s", strings.Join(extraFlags, " "))
			}

			for _, warning := range warnings {
				log.Warn(warning)
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

func parseEnvironmentFlags(f factory.Factory) ([]string, error) {
	// new environment flags parser
	flagsParser := f.NewEnvironmentFlagsParser()

	// parse other commands
	supportedCommands := []string{"", "analyze", "attach", "build", "deploy", "dev", "enter", "init", "login", "logs", "open", "print", "purge", "render", "run", "sync", "ui"}
	for _, command := range supportedCommands {
		err := flagsParser.Parse(command)
		if err != nil {
			return nil, errors.Wrap(err, "parse flags for command "+command)
		}
	}

	// apply flags
	return flagsParser.Apply(), nil
}
