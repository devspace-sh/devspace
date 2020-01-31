package cmd

import (
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	logger "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	varspkg "github.com/devspace-cloud/devspace/pkg/util/vars"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

// PrintCmd is a struct that defines a command call for "print"
type PrintCmd struct {
	*flags.GlobalFlags

	SkipInfo bool
}

// NewPrintCmd creates a new devspace print command
func NewPrintCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PrintCmd{GlobalFlags: globalFlags}

	printCmd := &cobra.Command{
		Use:   "print",
		Short: "Print displays the configuration",
		Long: `
#######################################################
################## devspace print #####################
#######################################################
Prints the configuration for the current or given 
profile after all patching and variable substitution
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	printCmd.Flags().BoolVar(&cmd.SkipInfo, "skip-info", false, "When enabled, only prints the configuration without additional information")

	return printCmd
}

// Run executes the command logic
func (cmd *PrintCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(configOptions, log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load config
	loadedConfig, err := configLoader.Load()
	if err != nil {
		return err
	}

	bsConfig, err := yaml.Marshal(loadedConfig)
	if err != nil {
		return err
	}

	if !cmd.SkipInfo {
		printExtraInfo(configOptions, configLoader, log)
		if err != nil {
			return err
		}
	}

	log.WriteString(string(bsConfig))

	return nil
}

func printExtraInfo(configOptions *loader.ConfigOptions, configLoader loader.ConfigLoader, log logger.Logger) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := constants.DefaultConfigPath
	if configOptions.ConfigPath != "" {
		path = configOptions.ConfigPath
	}
	absPath := filepath.Join(pwd, path)

	log.WriteString("\n-------------------\n\nVars:\n")

	headerColumnNames := []string{"Name", "Value"}
	values := [][]string{}

	err = fillCurrentVars(configOptions, configLoader, path, &values, log)
	if err != nil {
		return err
	}

	if len(values) > 0 {
		logger.PrintTable(log, headerColumnNames, values)
	} else {
		log.Info("No vars found")
	}

	log.WriteString("\n-------------------\n\nLoaded path: " + absPath + "\n\n-------------------\n\n")

	return nil
}

func varMatchFn(path, key, value string) bool {
	return varspkg.VarMatchRegex.MatchString(value)
}

func varReplaceFn(path, value string, generatedConfig *generated.Config, cmdVars map[string]string, configLoader loader.ConfigLoader, values *[][]string, log log.Logger) (interface{}, error) {
	return varspkg.ParseString(value, func(v string) (string, error) {
		isExists := false
		for _, value := range *values {
			if value[0] == v {
				isExists = true
			}
		}

		x, err := configLoader.ResolveVar(v, generatedConfig, cmdVars)
		if err != nil {
			return "", err
		}
		if !isExists {
			*values = append(*values, []string{
				v,
				x,
			})
		}

		return x, err
	})
}

func fillCurrentVars(configOptions *loader.ConfigOptions, configLoader loader.ConfigLoader, path string, values *[][]string, log logger.Logger) error {
	rawMap, err := configLoader.LoadRaw(path)

	// Get profile
	profile, err := versions.ParseProfile(rawMap, configOptions.Profile)
	if err != nil {
		return err
	}

	// Now delete not needed parts from config
	delete(rawMap, "vars")
	delete(rawMap, "profiles")
	delete(rawMap, "commands")

	// Apply profile
	if profile != nil {
		// Apply replace
		err = loader.ApplyReplace(rawMap, profile)
		if err != nil {
			return err
		}

		// Apply patches
		rawMap, err = loader.ApplyPatches(rawMap, profile)
		if err != nil {
			return err
		}
	}

	// Parse cli --var's
	cmdVars, err := loader.ParseVarsFromOptions(configOptions)
	if err != nil {
		return err
	}

	generatedConf, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Walk over data and fill in variables
	err = walk.Walk(rawMap, varMatchFn, func(path, value string) (interface{}, error) {
		return varReplaceFn(path, value, generatedConf, cmdVars, configLoader, values, log)
	})
	if err != nil {
		return err
	}

	return nil
}
