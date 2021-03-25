package set

import (
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type varCmd struct{}

func newVarCmd(f factory.Factory) *cobra.Command {
	cmd := &varCmd{}

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

	return varsCmd
}

// RunSetVar executes the set var command logic
func (cmd *varCmd) RunSetVar(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configLoader := f.NewConfigLoader("")
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load generated config
	generatedConfig, err := configLoader.LoadGenerated(nil)
	if err != nil {
		return err
	}

	allowedVars, err := getPossibleVars(generatedConfig, configLoader, log)
	if err != nil {
		return errors.Wrap(err, "get possible vars")
	}

	// Set vars
	for _, v := range args {
		if v == "" {
			continue
		}

		splitted := strings.SplitN(v, "=", 2)
		if len(splitted) < 2 {
			return errors.Errorf("Unexpected variable format. Expected key=value, got %s", v)
		} else if allowedVars[splitted[0]] == false {
			allowedVarsArr := []string{}
			for k := range allowedVars {
				allowedVarsArr = append(allowedVarsArr, k)
			}

			return errors.Errorf("Variable %s is not allowed. Allowed vars: %+v", splitted[0], allowedVarsArr)
		}

		generatedConfig.Vars[splitted[0]] = splitted[1]
	}

	// Save the config
	err = configLoader.SaveGenerated(generatedConfig)
	if err != nil {
		return errors.Errorf("Error saving config: %v", err)
	}

	log.Done("Successfully changed variables")
	return nil
}

func getPossibleVars(generatedConfig *generated.Config, configLoader loader.ConfigLoader, log log.Logger) (map[string]bool, error) {
	// Load variables
	rawMap, err := configLoader.LoadRaw()
	if err != nil {
		return nil, err
	}

	// Load defined variables
	vars, err := versions.ParseVariables(rawMap, log)
	if err != nil {
		return nil, err
	}

	retMap := make(map[string]bool)
	for varName := range generatedConfig.Vars {
		retMap[varName] = true
	}
	for _, v := range vars {
		retMap[v.Name] = true
	}

	return retMap, nil
}
