package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	v1 "github.com/covexo/devspace/pkg/devspace/config/v1"
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
	Namespace       string
	KubeContext     string
	Config          string
	ConfigOverwrite string
	DockerTarget    string
	CloudTarget     string
	SwitchContext   bool
	SkipBuild       bool
	GitBranch       string
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
devspace deploy --config=.devspace/deploy.yaml
devspace deploy --cloud-target=production
devspace deploy https://github.com/covexo/devspace --branch test
#######################################################`,
		Args: cobra.RangeArgs(0, 2),
		Run:  cmd.Run,
	}

	cobraCmd.Flags().StringVar(&cmd.flags.Namespace, "namespace", "", "The namespace to deploy to")
	cobraCmd.Flags().StringVar(&cmd.flags.KubeContext, "kube-context", "", "The kubernetes context to use for deployment")
	cobraCmd.Flags().StringVar(&cmd.flags.Config, "config", configutil.ConfigPath, "The devspace config file to load (default: '.devspace/config.yaml'")
	cobraCmd.Flags().StringVar(&cmd.flags.ConfigOverwrite, "config-overwrite", configutil.OverwriteConfigPath, "The devspace config overwrite file to load (default: '.devspace/overwrite.yaml'")
	cobraCmd.Flags().StringVar(&cmd.flags.DockerTarget, "docker-target", "", "The docker target to use for building")
	cobraCmd.Flags().StringVar(&cmd.flags.CloudTarget, "cloud-target", "", "When using a cloud provider, the target to use")
	cobraCmd.Flags().BoolVar(&cmd.flags.SwitchContext, "switch-context", true, "Switches the kube context to the deploy context")
	cobraCmd.Flags().BoolVar(&cmd.flags.SkipBuild, "skip-build", false, "Skips the image build & push step")
	// cobraCmd.Flags().StringVar(&cmd.flags.GitBranch, "branch", "master", "The git branch to checkout")

	rootCmd.AddCommand(cobraCmd)
}

// Run executes the down command logic
func (cmd *DeployCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()

	// Prepare the config
	cmd.prepareConfig()

	log.Infof("Loading config %s with overwrite config %s", configutil.ConfigPath, configutil.OverwriteConfigPath)

	// Configure cloud provider
	err := cloud.Configure(true, log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to configure cloud provider: %v", err)
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

	if cmd.flags.SkipBuild == false {
		// Force image build
		_, err = image.BuildAll(client, generatedConfig, true, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}
	}

	// Force deployment of all defined deployments
	err = deploy.All(client, generatedConfig, true, false, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Print domain name if we use a cloud provider
	config := configutil.GetConfig()
	cloudTarget := configutil.GetCurrentCloudTarget(config)
	if cloudTarget != nil {
		if generatedConfig != nil && generatedConfig.Cloud != nil && generatedConfig.Cloud.Targets != nil && generatedConfig.Cloud.Targets[*cloudTarget] != nil && generatedConfig.Cloud.Targets[*cloudTarget].Domain != nil {
			log.Infof("Your DevSpace is now reachable via ingress on this URL: http://%s", *generatedConfig.Cloud.Targets[*cloudTarget].Domain)
			log.Info("See https://devspace-cloud.com/domain-guide for more information")
		}
	}

	log.Donef("Successfully deployed!")
}

func (cmd *DeployCmd) prepareConfig() {
	if configutil.ConfigPath != cmd.flags.Config {
		configutil.ConfigPath = cmd.flags.Config

		// Don't use overwrite config if we use a different config
		configutil.OverwriteConfigPath = ""
	}
	if configutil.OverwriteConfigPath != cmd.flags.ConfigOverwrite {
		configutil.OverwriteConfigPath = cmd.flags.ConfigOverwrite
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
	if cmd.flags.CloudTarget != "" {
		config.Cluster.CloudTarget = &cmd.flags.CloudTarget
	}

	// Set defaults now
	configutil.SetDefaultsOnce()
}
