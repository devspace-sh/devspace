package cmd

import (
	"fmt"
	"os"

	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"
	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	"github.com/covexo/devspace/pkg/devspace/config"
	"github.com/covexo/devspace/pkg/util/logutil"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/spf13/cobra"
	"k8s.io/helm/pkg/helm"
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
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

// Run executes the down command logic
func (cmd *DownCmd) Run(cobraCmd *cobra.Command, args []string) {
	var err error

	log = logutil.GetLogger("default", true)
	privateConfig := &v1.PrivateConfig{}
	privateConfigExists, _ := config.ConfigExists(privateConfig)

	if !privateConfigExists {
		logutil.PrintFailMessage(fmt.Sprintf("Unable to load release name. Does the file .devspace/private.yaml exist?"), os.Stderr)
		return
	}

	err = config.LoadConfig(privateConfig)

	if err != nil {
		logutil.PrintFailMessage(fmt.Sprintf("Unable to load release name: %s. Does the file .devspace/private.yaml exist?", err.Error()), os.Stderr)
		return
	}

	releaseName := privateConfig.Release.Name
	kubectl, err := kubectl.NewClient()

	if err != nil {
		logutil.PrintFailMessage(fmt.Sprintf("Unable to create new kubectl client: %s", err.Error()), os.Stderr)
		return
	}

	client, err := helmClient.NewClient(kubectl, false)

	if err != nil {
		logutil.PrintFailMessage(fmt.Sprintf("Unable to initialize helm client: %s", err.Error()), os.Stderr)
		return
	}

	loadingText := logutil.NewLoadingText("Deleting release "+releaseName, os.Stdout)
	res, err := client.Client.DeleteRelease(releaseName, helm.DeletePurge(true))

	loadingText.Done()

	if res != nil && res.Info != "" {
		logutil.PrintDoneMessage(fmt.Sprintf("Successfully deleted release %s: %s", releaseName, res.Info), os.Stdout)
	} else if err != nil {
		logutil.PrintFailMessage(fmt.Sprintf("Error deleting release %s: %s", releaseName, err.Error()), os.Stdout)
	} else {
		logutil.PrintDoneMessage(fmt.Sprintf("Successfully deleted release %s", releaseName), os.Stdout)
	}
}
