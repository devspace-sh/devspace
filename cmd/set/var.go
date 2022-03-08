package set

import (
	"context"
	"strings"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type varCmd struct {
	*flags.GlobalFlags

	Overwrite bool
}

func newVarCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &varCmd{GlobalFlags: globalFlags}

	varsCmd := &cobra.Command{
		Use:   "var",
		Short: "Sets a variable",
		Long: `
#######################################################
################# devspace set var ####################
#######################################################
Sets a specific variable 

Examples:
devspace set var key=value
devspace set var key=value key2=value2
#######################################################
	`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunSetVar(f, cobraCmd, args)
		}}

	varsCmd.Flags().BoolVar(&cmd.Overwrite, "overwrite", true, "If true will overwrite the variables value even if its set already")
	return varsCmd
}

// RunSetVar executes the set var command logic
func (cmd *varCmd) RunSetVar(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load config and find all variables in it
	variableParser := &variableParser{}
	c, err := configLoader.LoadWithParser(context.Background(), nil, nil, variableParser, cmd.ToConfigOptions(), log)
	if err != nil {
		return err
	}

	// Set vars
	for _, v := range args {
		if v == "" {
			continue
		}

		// check if variable can be set
		splitted := strings.SplitN(v, "=", 2)
		if len(splitted) < 2 {
			return errors.Errorf("Unexpected variable format. Expected key=value, got %s", v)
		} else if variable.IsPredefinedVariable(splitted[0]) {
			return errors.Errorf("cannot set predefined variable %s", splitted[0])
		}

		found := false
		for _, u := range variableParser.Used {
			if u.Name == splitted[0] {
				found = true
				break
			}
		}
		if !found {
			allowedVarsArr := []string{}
			for _, v := range variableParser.Used {
				allowedVarsArr = append(allowedVarsArr, v.Name)
			}

			return errors.Errorf("variable %s is not allowed. Allowed vars: %+v", splitted[0], allowedVarsArr)
		}

		// try to find it in definitions
		for _, def := range variableParser.Definitions {
			if def.Name == splitted[0] {
				if def.Command != "" || len(def.Commands) > 0 || def.Source == latest.VariableSourceCommand || def.Source == latest.VariableSourceEnv || def.Source == latest.VariableSourceNone {
					return errors.Errorf("cannot set variable %s, because variable is not loaded from cache. Please change variable type to cache it", def.Name)
				}
			}
		}

		// only overwrite it if the flag is true and value is not set yet
		_, found = c.LocalCache().GetVar(splitted[0])
		if cmd.Overwrite || !found {
			c.LocalCache().SetVar(splitted[0], splitted[1])
		} else {
			log.Infof("Skip variable %s, because it already has a value", splitted[0])
		}
	}

	// Save the config
	err = c.LocalCache().Save()
	if err != nil {
		return errors.Errorf("Error saving config: %v", err)
	}

	log.Done("Successfully changed variables")
	return nil
}

type variableParser struct {
	Definitions map[string]*latest.Variable
	Used        []*latest.Variable
}

func (v *variableParser) Parse(ctx context.Context, originalRawConfig map[string]interface{}, rawConfig map[string]interface{}, resolver variable.Resolver, log log.Logger) (*latest.Config, map[string]interface{}, error) {
	// Find out what vars are really used
	varsUsed, err := resolver.FindVariables(rawConfig)
	if err != nil {
		return nil, nil, err
	}

	v.Definitions = resolver.DefinedVars()
	v.Used = varsUsed
	return latest.NewRaw(), map[string]interface{}{}, nil
}
