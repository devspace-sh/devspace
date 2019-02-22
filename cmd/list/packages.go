package list

import (
	"os"
	"path/filepath"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/yamlutil"
	"github.com/spf13/cobra"
)

type packagesCmd struct{}

func newPackagesCmd() *cobra.Command {
	cmd := &packagesCmd{}

	packagesCmd := &cobra.Command{
		Use:   "packages",
		Short: "Lists all added packages",
		Long: `
	#######################################################
	############### devspace list packages ################
	#######################################################
	Lists the packages that were added to the DevSpace
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunListPackage,
	}

	return packagesCmd
}

// RunListPackage runs the list sync command logic
func (cmd *packagesCmd) RunListPackage(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	headerColumnNames := []string{
		"Name",
		"Version",
		"Repository",
	}
	values := [][]string{}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	requirementsFile := filepath.Join(cwd, "chart", "requirements.yaml")
	_, err = os.Stat(requirementsFile)
	if os.IsNotExist(err) == false {
		yamlContents := map[interface{}]interface{}{}
		err = yamlutil.ReadYamlFromFile(requirementsFile, yamlContents)
		if err != nil {
			log.Fatalf("Error parsing %s: %v", requirementsFile, err)
		}

		if dependencies, ok := yamlContents["dependencies"]; ok {
			if dependenciesArr, ok := dependencies.([]interface{}); ok {
				for _, dependency := range dependenciesArr {
					if dependencyMap, ok := dependency.(map[interface{}]interface{}); ok {
						values = append(values, []string{
							dependencyMap["name"].(string),
							dependencyMap["version"].(string),
							dependencyMap["repository"].(string),
						})
					}
				}
			}
		}
	}

	log.PrintTable(headerColumnNames, values)
}
