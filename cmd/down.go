package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	helmClient "github.com/covexo/devspace/pkg/devspace/deploy/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"

	"github.com/spf13/cobra"
)

// DownCmd holds the required data for the down cmd
type DownCmd struct {
	flags *DownCmdFlags
}

// DownCmdFlags holds the possible down cmd flags
type DownCmdFlags struct {
}

func init() {
	cmd := &DownCmd{
		flags: &DownCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "down",
		Short: "Shutdown your DevSpace",
		Long: `
#######################################################
################### devspace down #####################
#######################################################
Stops your DevSpace by removing the release via helm.
If you want to remove all DevSpace related data from
your project, use: devspace reset
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

// Run executes the down command logic
func (cmd *DownCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()

	config := configutil.GetConfig(false)

	releaseName := *config.DevSpace.Release.Name
	kubectl, err := kubectl.NewClient()

	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %s", err.Error())
	}

	client, err := helmClient.NewClient(kubectl, false)

	if err != nil {
		log.Fatalf("Unable to initialize helm client: %s", err.Error())
	}

	log.StartWait("Deleting release " + releaseName)
	res, err := client.DeleteRelease(releaseName, true)
	log.StopWait()

	if res != nil && res.Info != "" {
		log.Donef("Successfully deleted release %s: %s", releaseName, res.Info)
	} else if err != nil {
		log.Donef("Error deleting release %s: %s", releaseName, err.Error())
	} else {
		log.Donef("Successfully deleted release %s", releaseName)
	}
}
