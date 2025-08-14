package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"sort"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logger "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

// PrintCmd is a struct that defines a command call for "print"
type PrintCmd struct {
	*flags.GlobalFlags

	Out        io.Writer
	StripNames bool
	SkipInfo   bool

	Dependency string
}

// NewPrintCmd creates a new devspace print command
func NewPrintCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PrintCmd{
		GlobalFlags: globalFlags,
		StripNames:  true,
		Out:         os.Stdout,
	}

	printCmd := &cobra.Command{
		Use:   "print",
		Short: "Prints displays the configuration",
		Long: `
#######################################################
################## devspace print #####################
#######################################################
Prints the configuration for the current or given 
profile after all patching and variable substitution
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	printCmd.Flags().BoolVar(&cmd.SkipInfo, "skip-info", false, "When enabled, only prints the configuration without additional information")
	printCmd.Flags().StringVar(&cmd.Dependency, "dependency", "", "The dependency to print the config from. Use dot to access nested dependencies (e.g. dep1.dep2)")

	return printCmd
}

// Run executes the command logic
func (cmd *PrintCmd) Run(f factory.Factory) error {
	// Set config root
	log := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	} else if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		log.Warnf("Unable to create new kubectl client: %v", err)
	}

	parser := loader.NewEagerParser()

	// load config
	config, err := configLoader.LoadWithParser(context.Background(), nil, client, parser, configOptions, log)
	if err != nil {
		return err
	}

	// create devspace context
	ctx := devspacecontext.NewContext(context.Background(), config.Variables(), log).
		WithConfig(config).
		WithKubeClient(client)

	// resolve dependencies
	dependencies, err := dependency.NewManagerWithParser(ctx, configOptions, parser).ResolveAll(ctx, dependency.ResolveOptions{})
	if err != nil {
		log.Warnf("Error resolving dependencies: %v", err)
	}
	ctx = ctx.WithDependencies(dependencies)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "print")
	if err != nil {
		return err
	}

	if cmd.Dependency != "" {
		dep := dependency.GetDependencyByPath(dependencies, cmd.Dependency)
		if dep == nil {
			return fmt.Errorf("couldn't find dependency %s: make sure it gets loaded correctly", cmd.Dependency)
		}

		ctx = ctx.AsDependency(dep)
	}

	bsConfig, err := marshalConfig(ctx.Config().Config(), cmd.StripNames)
	if err != nil {
		return err
	}

	if !cmd.SkipInfo {
		err = printExtraInfo(ctx.Config(), dependencies, log)
		if err != nil {
			return err
		}
	}

	if cmd.Out != nil {
		_, err := cmd.Out.Write(bsConfig)
		if err != nil {
			return err
		}
	} else {
		log.WriteString(logrus.InfoLevel, string(bsConfig))
	}

	return nil
}

func marshalConfig(config *latest.Config, stripNames bool) ([]byte, error) {
	// remove the auto generated names
	if stripNames {
		for k := range config.Images {
			config.Images[k].Name = ""
		}
		for k := range config.Deployments {
			config.Deployments[k].Name = ""
		}
		for k := range config.Dependencies {
			config.Dependencies[k].Name = ""
		}
		for k := range config.Pipelines {
			config.Pipelines[k].Name = ""
		}
		for k := range config.Dev {
			config.Dev[k].Name = ""
			for c := range config.Dev[k].Containers {
				config.Dev[k].Containers[c].Container = ""
			}
		}
		for k := range config.Vars {
			config.Vars[k].Name = ""
		}
		for k := range config.PullSecrets {
			config.PullSecrets[k].Name = ""
		}
		for k := range config.Commands {
			config.Commands[k].Name = ""
		}
	}

	return yaml.Marshal(config)
}

func printExtraInfo(config config.Config, dependencies []types.Dependency, log logger.Logger) error {
	log.WriteString(logrus.InfoLevel, "\n-------------------\n\nVars:\n")

	headerColumnNames := []string{"Name", "Value"}
	values := [][]string{}
	resolvedVars := config.Variables()
	for varName, varValue := range resolvedVars {
		values = append(values, []string{
			varName,
			fmt.Sprintf("%v", varValue),
		})
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i][0] < values[j][0]
	})

	if len(values) > 0 {
		logger.PrintTable(log, headerColumnNames, values)
	} else {
		log.Info("No vars found")
	}

	log.WriteString(logrus.InfoLevel, "\n-------------------\n\nLoaded path: "+config.Path()+"\n\n-------------------\n\n")

	if len(dependencies) > 0 {
		log.WriteString(logrus.InfoLevel, "Dependency Tree:\n\n> Root\n")
		for _, dep := range dependencies {
			printDependencyRecursive("--", dep, 5, log)
		}
		log.WriteString(logrus.InfoLevel, "\n-------------------\n\n")
	}

	return nil
}

func printDependencyRecursive(prefix string, dep types.Dependency, maxDepth int, log logger.Logger) {
	if maxDepth == 0 {
		return
	}
	log.WriteString(logrus.InfoLevel, prefix+"> "+dep.Name()+"\n")
	for _, child := range dep.Children() {
		printDependencyRecursive(prefix+"--", child, maxDepth-1, log)
	}
}
