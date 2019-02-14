package cmd

import (
	"fmt"
	"strings"

	"github.com/covexo/devspace/cmd/add"
	"github.com/covexo/devspace/cmd/create"
	"github.com/covexo/devspace/cmd/list"
	"github.com/covexo/devspace/cmd/remove"
	"github.com/covexo/devspace/cmd/status"
	"github.com/covexo/devspace/cmd/use"
	"github.com/covexo/devspace/pkg/devspace/upgrade"
	"github.com/covexo/devspace/pkg/util/log"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "devspace",
	Short: "Welcome to the DevSpace CLI!",
	Long: `With a DevSpace you can program, build and execute cloud-native applications
	 directly inside a Kubernetes cluster. Start your DevSpace now with:

	devspace up`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if upgrade.GetVersion() != "" && strings.Contains(upgrade.GetVersion(), "-alpha") == false && strings.Contains(upgrade.GetVersion(), "-beta") == false {
		rootCmd.Version = upgrade.GetVersion()
		newerVersion, err := upgrade.CheckForNewerVersion()

		if err == nil && newerVersion != "" {
			log.Warnf("There is a newer version of devspace cli v%s. Run `devspace upgrade` to update the cli.\n", newerVersion)
		} else if err != nil {
			log.Warnf("Couldn't check for newest version: %s\n", err.Error())
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func init() {
	rootCmd.AddCommand(add.NewAddCmd())
	rootCmd.AddCommand(create.NewCreateCmd())
	rootCmd.AddCommand(list.NewListCmd())
	rootCmd.AddCommand(remove.NewRemoveCmd())
	rootCmd.AddCommand(status.NewStatusCmd())
	rootCmd.AddCommand(use.NewUseCmd())

	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
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
