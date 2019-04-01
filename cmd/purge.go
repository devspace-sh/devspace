package cmd

import (
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/spf13/cobra"
)

// PurgeCmd holds the required data for the purge cmd
type PurgeCmd struct {
	flags *PurgeCmdFlags
}

// PurgeCmdFlags holds the possible down cmd flags
type PurgeCmdFlags struct {
	deployment string
}

func init() {
	cmd := &PurgeCmd{
		flags: &PurgeCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete all deployed kubernetes resources",
		Long: `
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kuberenetes resources. 
Warning: will delete everything that is defined in the 
local chart, including persistent volume claims!
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}

	cobraCmd.Flags().StringVarP(&cmd.flags.deployment, "deployment", "d", "", "The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)")

	rootCmd.AddCommand(cobraCmd)
}

// Run executes the purge command logic
func (cmd *PurgeCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	log.StartFileLogging()

	kubectl, err := kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %s", err.Error())
	}

	deployments := []string{}
	if cmd.flags.deployment != "" {
		deployments = strings.Split(cmd.flags.deployment, ",")
		for index := range deployments {
			deployments[index] = strings.TrimSpace(deployments[index])
		}
	}

	deploy.PurgeDeployments(kubectl, deployments)
}
