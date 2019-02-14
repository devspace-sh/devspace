package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/devspace/deploy"
	"github.com/covexo/devspace/pkg/devspace/docker"
	"github.com/covexo/devspace/pkg/devspace/image"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/devspace/registry"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// DeployCmd holds the required data for the down cmd
type DeployCmd struct {
	flags *DeployCmdFlags
}

// DeployCmdFlags holds the possible down cmd flags
type DeployCmdFlags struct {
	Namespace     string
	KubeContext   string
	Config        string
	DockerTarget  string
	SwitchContext bool
	SkipBuild     bool
}

func init() {
	cmd := &DeployCmd{
		flags: &DeployCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy your DevSpace to a target cluster",
		Long: `
#######################################################
################## devspace deploy ####################
#######################################################
Deploys the devspace to a target cluster:

devspace deploy --namespace=deploy
devspace deploy --namespace=deploy --docker-target=production
devspace deploy --kube-context=deploy-context
#######################################################`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.Run,
	}

	cobraCmd.Flags().StringVar(&cmd.flags.Namespace, "namespace", "", "The namespace to deploy to")
	cobraCmd.Flags().StringVar(&cmd.flags.KubeContext, "kube-context", "", "The kubernetes context to use for deployment")
	cobraCmd.Flags().StringVar(&cmd.flags.Config, "config", configutil.ConfigPath, "The devspace config file to load (default: '.devspace/config.yaml'")
	cobraCmd.Flags().StringVar(&cmd.flags.DockerTarget, "docker-target", "", "The docker target to use for building")
	cobraCmd.Flags().BoolVar(&cmd.flags.SwitchContext, "switch-context", false, "Switches the kube context to the deploy context")

	rootCmd.AddCommand(cobraCmd)
}

// Run executes the down command logic
func (cmd *DeployCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	// Start file logging
	log.StartFileLogging()

	// Prepare the config
	cmd.prepareConfig()

	// Check if there is a space configured
	if len(args) > 0 {
		// Configure cloud provider
		err := cloud.ConfigureWithSpaceName(args[0], log.GetInstance())
		if err != nil {
			log.Fatalf("Unable to configure cloud provider: %v", err)
		}
	} else {
		// Configure cloud provider
		err := cloud.Configure(log.GetInstance())
		if err != nil {
			log.Fatalf("Unable to configure cloud provider: %v", err)
		}
	}

	// Create kubectl client
	client, err := kubectl.NewClientWithContextSwitch(cmd.flags.SwitchContext)
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
	mustRedeploy, err := image.BuildAll(client, generatedConfig, false, false, log.GetInstance())
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

	// Force deployment of all defined deployments
	err = deploy.All(client, generatedConfig, false, mustRedeploy, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Print domain name if we use a cloud provider
	config := configutil.GetConfig()
	if config.Cluster != nil && config.Cluster.CloudProvider != nil {
		generatedConfig, _ := generated.LoadConfig()
		if generatedConfig != nil && generatedConfig.Space != nil && generatedConfig.Space.Domain != nil {
			log.Infof("The Space is now reachable via ingress on this URL: https://%s", *generatedConfig.Space.Domain)
		}
	}

	log.Donef("Successfully deployed!")
}

func (cmd *DeployCmd) prepareConfig() {
	if configutil.ConfigPath != cmd.flags.Config {
		configutil.ConfigPath = cmd.flags.Config
	}

	// Load Config and modify it
	config := configutil.GetConfigWithoutDefaults(true)

	if cmd.flags.Namespace != "" {
		config.Cluster = &v1.Cluster{
			Namespace:   &cmd.flags.Namespace,
			KubeContext: config.Cluster.KubeContext,
			APIServer:   config.Cluster.APIServer,
			CaCert:      config.Cluster.CaCert,
			User:        config.Cluster.User,
		}

		log.Infof("Using %s namespace for deploying", cmd.flags.Namespace)
	}
	if cmd.flags.KubeContext != "" {
		config.Cluster = &v1.Cluster{
			Namespace:   config.Cluster.Namespace,
			KubeContext: &cmd.flags.KubeContext,
			APIServer:   config.Cluster.APIServer,
			CaCert:      config.Cluster.CaCert,
			User:        config.Cluster.User,
		}

		log.Infof("Using %s kube context for deploying", cmd.flags.KubeContext)
	}
	if cmd.flags.DockerTarget != "" {
		if config.Images != nil {
			for _, imageConf := range *config.Images {
				if imageConf.Build == nil {
					imageConf.Build = &v1.BuildConfig{}
				}
				if imageConf.Build.Options == nil {
					imageConf.Build.Options = &v1.BuildOptions{}
				}
				imageConf.Build.Options.Target = &cmd.flags.DockerTarget
			}
		}
	}

	// Set defaults now
	configutil.ValidateOnce()
}
