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
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
)

type DownCmd struct {
	flags         *DownCmdFlags
	helm          *helmClient.HelmClientWrapper
	kubectl       *kubernetes.Clientset
	privateConfig *v1.PrivateConfig
	dsConfig      *v1.DevSpaceConfig
	workdir       string
}

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

func (cmd *DownCmd) Run(cobraCmd *cobra.Command, args []string) {
	var err error

	log = logutil.GetLogger("default", true)
	workdir, workdirErr := os.Getwd()

	if workdirErr != nil {
		logutil.PrintFailMessage(fmt.Sprintf("Unable to determine current workdir: %s", workdirErr.Error()), os.Stderr)
		return
	}

	cmd.workdir = workdir
	cmd.privateConfig = &v1.PrivateConfig{}
	cmd.dsConfig = &v1.DevSpaceConfig{}

	privateConfigExists, _ := config.ConfigExists(cmd.privateConfig)
	dsConfigExists, _ := config.ConfigExists(cmd.dsConfig)

	if !privateConfigExists || !dsConfigExists {
		initCmd := &InitCmd{
			flags: InitCmdFlagsDefault,
		}
		initCmd.Run(nil, []string{})
	}

	config.LoadConfig(cmd.privateConfig)
	config.LoadConfig(cmd.dsConfig)

	releaseName := cmd.privateConfig.Release.Name
	cmd.kubectl, err = kubectl.NewClient()

	if err != nil {
		logutil.PrintFailMessage(fmt.Sprintf("Unable to create new kubectl client: %s", err.Error()), os.Stderr)
		return
	}

	client, helmErr := helmClient.NewClient(cmd.kubectl, false)

	if helmErr != nil {
		logutil.PrintFailMessage(fmt.Sprintf("Unable to initialize helm client: %s", helmErr.Error()), os.Stderr)
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
