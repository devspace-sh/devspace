package cmd

import (
	"context"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/spf13/cobra"
)

// PurgeCmd holds the required data for the purge cmd
type PurgeCmd struct {
	Deployments             string
	AllowCyclicDependencies bool
	PurgeDependencies       bool

	Namespace   string
	KubeContext string
}

// NewPurgeCmd creates a new purge command
func NewPurgeCmd() *cobra.Command {
	cmd := &PurgeCmd{}

	purgeCmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete deployed resources",
		Long: `
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kuberenetes resources:

devspace purge
devspace purge --dependencies
devspace purge -d my-deployment
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}

	purgeCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "The namespace to purge the deployments from")
	purgeCmd.Flags().StringVar(&cmd.KubeContext, "kube-context", "", "The kubernetes context to use")

	purgeCmd.Flags().StringVarP(&cmd.Deployments, "deployments", "d", "", "The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)")
	purgeCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")
	purgeCmd.Flags().BoolVar(&cmd.PurgeDependencies, "dependencies", false, "When enabled purges the dependencies as well")

	return purgeCmd
}

// Run executes the purge command logic
func (cmd *PurgeCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	log.StartFileLogging()

	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, false)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	err = client.PrintWarning(false, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Signal that we are working on the space if there is any
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Get config with adjusted cluster config
	config := configutil.GetConfig(context.WithValue(context.Background(), constants.KubeContextKey, client.CurrentContext))

	deployments := []string{}
	if cmd.Deployments != "" {
		deployments = strings.Split(cmd.Deployments, ",")
		for index := range deployments {
			deployments[index] = strings.TrimSpace(deployments[index])
		}
	}

	// Purge deployments
	deploy.PurgeDeployments(config, generatedConfig.GetActive(), client, deployments, log.GetInstance())

	// Purge dependencies
	if cmd.PurgeDependencies {
		err = dependency.PurgeAll(config, generatedConfig, client, cmd.AllowCyclicDependencies, log.GetInstance())
		if err != nil {
			log.Errorf("Error purging dependencies: %v", err)
		}
	}

	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Errorf("Error saving generated.yaml: %v", err)
	}
}
