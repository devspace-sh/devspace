package use

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type contextCmd struct{}

func newContextCmd() *cobra.Command {
	cmd := &contextCmd{}

	return &cobra.Command{
		Use:   "context",
		Short: "Change current kubectl context to space",
		Long: `
#######################################################
################ devspace use context #################
#######################################################
Change the current kubectl context to the defined space
kubernetes credentials

Example:
devspace use context         // Use active space credentials
devspace use context myspace // Use different space credentials
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunUseContext,
	}
}

// RunUseConfig executes the "devspace use config" command logic
func (*contextCmd) RunUseContext(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	var space *generated.SpaceConfig
	if len(args) == 0 {
		if configExists == false {
			log.Fatal("No space configured")
		}

		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading generated.yaml: %v", err)
		}
		if generatedConfig.Space == nil {
			log.Fatal("No space configured. Run `devspace use space` to configure space for active project")
		}

		space = generatedConfig.Space
	} else {
		provider, err := cloud.GetCurrentProvider(log.GetInstance())
		if err != nil {
			log.Fatalf("Error getting provider config: %v", err)
		}

		space, err = provider.GetSpaceByName(args[0])
		if err != nil {
			log.Fatalf("Error retrieving space: %v", err)
		}
	}

	err = cloud.UpdateKubeConfig(cloud.GetKubeContextNameFromSpace(space), space, true)
	if err != nil {
		log.Fatalf("Error saving kube config: %v", err)
	}

	log.Infof("Successfully changed kubectl context to space %s", space.Name)
}
