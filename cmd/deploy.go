package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/deploy"
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
	CloudTarget   string
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
devspace deploy --config=.devspace/deploy.yaml
devspace deploy --cloud-target=production
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}

	cobraCmd.Flags().StringVar(&cmd.flags.Namespace, "namespace", "", "The namespace to deploy to")
	cobraCmd.Flags().StringVar(&cmd.flags.KubeContext, "kube-context", "", "The kubernetes context to use for deployment")
	cobraCmd.Flags().StringVar(&cmd.flags.Config, "config", configutil.ConfigPath, "The devspace config file to load (default: '.devspace/config.yaml'")
	cobraCmd.Flags().StringVar(&cmd.flags.DockerTarget, "docker-target", "", "The docker target to use for building")
	cobraCmd.Flags().StringVar(&cmd.flags.CloudTarget, "cloud-target", "", "When using a cloud provider, the target to use")
	cobraCmd.Flags().BoolVar(&cmd.flags.SwitchContext, "switch-context", false, "Switches the kube context to the deploy context")
	cobraCmd.Flags().BoolVar(&cmd.flags.SkipBuild, "skip-build", false, "Skips the image build & push step")

	rootCmd.AddCommand(cobraCmd)
}

// Run executes the down command logic
func (cmd *DeployCmd) Run(cobraCmd *cobra.Command, args []string) {
	cloud.UseDeployTarget = true
	log.StartFileLogging()

	// Prepare the config
	cmd.prepareConfig()

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

	// Create pull secrets and private registry if necessary
	err = registry.InitRegistries(client, log.GetInstance())
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
	// TODO: Change this
	if cloud.DevSpaceURL != "" {
		log.Infof("Your DevSpace is now reachable via ingress on this URL: http://%s", cloud.DevSpaceURL)
		log.Info("See https://devspace-cloud.com/domain-guide for more information")
	}

	log.Donef("Successfully deployed!")
}

func (cmd *DeployCmd) prepareConfig() {
	if configutil.ConfigPath != cmd.flags.Config {
		configutil.ConfigPath = cmd.flags.Config

		// Don't use overwrite config if we use a different config
		configutil.OverwriteConfigPath = ""
	}

	// Load Config and modify it
	config := configutil.GetConfigWithoutDefaults()

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
		config.Cluster.CloudProviderDeployTarget = &cmd.flags.CloudTarget
	}

	// Set defaults now
	configutil.SetDefaultsOnce()
}
