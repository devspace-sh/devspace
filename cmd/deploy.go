package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/image"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

// DeployCmd holds the required data for the down cmd
type DeployCmd struct {
	Namespace     string
	KubeContext   string
	DockerTarget  string
	ForceBuild    bool
	ForceDeploy   bool
	SwitchContext bool
}

// NewDeployCmd creates a new deploy command
func NewDeployCmd() *cobra.Command {
	cmd := &DeployCmd{}

	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the project",
		Long: `
#######################################################
################## devspace deploy ####################
#######################################################
Deploys the current project to a Space or namespace:

devspace deploy
devspace deploy --namespace=deploy
devspace deploy --namespace=deploy
devspace deploy --kube-context=deploy-context
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}

	deployCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "The namespace to deploy to")
	deployCmd.Flags().StringVar(&cmd.KubeContext, "kube-context", "", "The kubernetes context to use for deployment")
	deployCmd.Flags().StringVar(&cmd.DockerTarget, "docker-target", "", "The docker target to use for building")

	deployCmd.Flags().BoolVar(&cmd.SwitchContext, "switch-context", false, "Switches the kube context to the deploy context")
	deployCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to (re-)build every image")
	deployCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to (re-)deploy every deployment")

	return deployCmd
}

// Run executes the down command logic
func (cmd *DeployCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Start file logging
	log.StartFileLogging()

	// Prepare the config
	cmd.prepareConfig()

	// Create kubectl client
	client, err := kubectl.NewClientWithContextSwitch(cmd.SwitchContext)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	// Create namespace if necessary
	err = kubectl.EnsureDefaultNamespace(client, log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to create namespace: %v", err)
	}

	// Create cluster binding if necessary
	err = kubectl.EnsureGoogleCloudClusterRoleBinding(client, log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to ensure cluster-admin role binding: %v", err)
	}

	// Create docker client
	dockerClient, err := docker.NewClient(false)

	// Create pull secrets and private registry if necessary
	err = registry.InitRegistries(dockerClient, client, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Load generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading generated.yaml: %v", err)
	}

	// Force image build
	mustRedeploy, err := image.BuildAll(client, generatedConfig, false, cmd.ForceBuild, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Save config if an image was built
	if mustRedeploy == true {
		err := generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatalf("Error saving generated config: %v", err)
		}
	}

	// Deploy all defined deployments
	err = deploy.All(client, generatedConfig, false, mustRedeploy || cmd.ForceDeploy, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Save Config
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatalf("Error saving generated config: %v", err)
	}

	if generatedConfig.CloudSpace != nil {
		// Create ingress if there is none
		err = cloud.CreateIngress(client)
		if err != nil {
			log.Warnf("Error creating ingress: %v", err)
		}

		log.Donef("Successfully deployed!")
		log.Infof("Run: \n- `%s` to open the app in the browser\n- `%s` to open a shell into the container\n- `%s` to show the container logs\n- `%s` to open the management ui\n- `%s` to analyze the space for potential issues", ansi.Color("devspace open", "white+b"), ansi.Color("devspace enter", "white+b"), ansi.Color("devspace logs", "white+b"), ansi.Color("devspace ui", "white+b"), ansi.Color("devspace analyze", "white+b"))
	} else {
		log.Donef("Successfully deployed!")
		log.Infof("Run `%s` to check for potential issues", ansi.Color("devspace analyze", "white+b"))
	}
}

func (cmd *DeployCmd) prepareConfig() {
	// Load Config and modify it
	config := configutil.GetConfigWithoutDefaults(true)

	if cmd.Namespace != "" {
		config.Cluster = &v1.Cluster{
			Namespace:   &cmd.Namespace,
			KubeContext: config.Cluster.KubeContext,
			APIServer:   config.Cluster.APIServer,
			CaCert:      config.Cluster.CaCert,
			User:        config.Cluster.User,
		}

		log.Infof("Using %s namespace for deploying", cmd.Namespace)
	}
	if cmd.KubeContext != "" {
		config.Cluster = &v1.Cluster{
			Namespace:   config.Cluster.Namespace,
			KubeContext: &cmd.KubeContext,
			APIServer:   config.Cluster.APIServer,
			CaCert:      config.Cluster.CaCert,
			User:        config.Cluster.User,
		}

		log.Infof("Using %s kube context for deploying", cmd.KubeContext)
	}
	if cmd.DockerTarget != "" {
		if config.Images != nil {
			for _, imageConf := range *config.Images {
				if imageConf.Build == nil {
					imageConf.Build = &v1.BuildConfig{}
				}
				if imageConf.Build.Options == nil {
					imageConf.Build.Options = &v1.BuildOptions{}
				}
				imageConf.Build.Options.Target = &cmd.DockerTarget
			}
		}
	}

	// Set defaults now
	configutil.ValidateOnce()
}
