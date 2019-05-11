package cmd

import (
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/spf13/cobra"
)

// PurgeCmd holds the required data for the purge cmd
type PurgeCmd struct {
	Deployments             string
	Namespace               string
	AllowCyclicDependencies bool
	PurgeDependencies       bool
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
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	log.StartFileLogging()

	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Errorf("Error loading generated.yaml: %v", err)
		return
	}

	// Get the config
	config := cmd.loadConfig(generatedConfig)

	kubectl, err := kubectl.NewClient(config)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	deployments := []string{}
	if cmd.Deployments != "" {
		deployments = strings.Split(cmd.Deployments, ",")
		for index := range deployments {
			deployments[index] = strings.TrimSpace(deployments[index])
		}
	}

	// Purge deployments
	deploy.PurgeDeployments(config, generatedConfig.GetActive(), kubectl, deployments, log.GetInstance())

	// Purge dependencies
	if cmd.PurgeDependencies {
		err = dependency.PurgeAll(config, generatedConfig, cmd.AllowCyclicDependencies, log.GetInstance())
		if err != nil {
			log.Errorf("Error purging dependencies: %v", err)
		}
	}

	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Errorf("Error saving generated.yaml: %v", err)
	}
}

func (cmd *PurgeCmd) loadConfig(generatedConfig *generated.Config) *latest.Config {
	// Load Config and modify it
	config, err := configutil.GetConfigFromPath(".", generatedConfig.ActiveConfig, true, generatedConfig)
	if err != nil {
		log.Fatal(err)
	}

	if cmd.Namespace != "" {
		config.Cluster = &v1.Cluster{
			Namespace:   &cmd.Namespace,
			KubeContext: config.Cluster.KubeContext,
			APIServer:   config.Cluster.APIServer,
			CaCert:      config.Cluster.CaCert,
			User:        config.Cluster.User,
		}

		log.Infof("Using %s namespace", cmd.Namespace)
	}

	return config
}
