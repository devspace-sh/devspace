package set

import (
	"io/ioutil"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

type varCmd struct{}

func newVarCmd() *cobra.Command {
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
		RunE: cmd.RunSetVar,
	}

	return varsCmd
}

// RunSetVar executes the set var command logic
func (cmd *varCmd) RunSetVar(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(nil, log.GetInstance())
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Load generated config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	allowedVars, err := getPossibleVars(generatedConfig, log.GetInstance())
	if err != nil {
		return errors.Wrap(err, "get possible vars")
	}

	// Set vars
	for _, v := range args {
		if v == "" {
			continue
		}

		splitted := strings.Split(v, "=")
		if len(splitted) != 2 {
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

	log.GetInstance().Done("Successfully changed variables")
	return nil
}

func getPossibleVars(generatedConfig *generated.Config, log log.Logger) (map[string]bool, error) {
	// Load variables
	bytes, err := ioutil.ReadFile(constants.DefaultConfigPath)
	if err != nil {
		return nil, err
	}
	rawMap := map[interface{}]interface{}{}
	err = yaml.Unmarshal(bytes, &rawMap)
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
