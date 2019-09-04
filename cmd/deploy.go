package cmd

import (
	"fmt"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

// DeployCmd holds the required data for the down cmd
type DeployCmd struct {
	Namespace   string
	KubeContext string

	DockerTarget string

	ForceBuild        bool
	SkipBuild         bool
	BuildSequential   bool
	ForceDeploy       bool
	Deployments       string
	ForceDependencies bool

	SwitchContext bool
	SkipPush      bool

	AllowCyclicDependencies bool
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

	deployCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")

	deployCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "The namespace to deploy to")
	deployCmd.Flags().StringVar(&cmd.KubeContext, "kube-context", "", "The kubernetes context to use for deployment")

	deployCmd.Flags().BoolVar(&cmd.SwitchContext, "switch-context", true, "Switches the kube context to the deploy context")
	deployCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")

	deployCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to (re-)build every image")
	deployCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	deployCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	deployCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to (re-)deploy every deployment")
	deployCmd.Flags().BoolVar(&cmd.ForceDependencies, "force-dependencies", false, "Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)")
	deployCmd.Flags().StringVar(&cmd.Deployments, "deployments", "", "Only deploy a specifc deployment (You can specify multiple deployments comma-separated")

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

	// Validate flags
	cmd.validateFlags()

	// Load generated config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading generated.yaml: %v", err)
	}

	// Create kubectl client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	// Warn the user if we deployed into a different context before
	err = client.PrintWarning(true, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Get config with adjusted cluster config
	config := configutil.GetConfig()

	// Signal that we are working on the space if there is any
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Create namespace if necessary
	err = client.EnsureDefaultNamespace(log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to create namespace: %v", err)
	}

	// Create cluster binding if necessary
	err = client.EnsureGoogleCloudClusterRoleBinding(log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to ensure cluster-admin role binding: %v", err)
	}

	// Create docker client
	dockerClient, err := docker.NewClient(log.GetInstance())
	if err != nil {
		dockerClient = nil
	}

	// Create pull secrets and private registry if necessary
	err = registry.CreatePullSecrets(config, client, dockerClient, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Dependencies
	err = dependency.DeployAll(config, generatedConfig, client, cmd.AllowCyclicDependencies, false, cmd.SkipPush, cmd.ForceDependencies, cmd.SkipBuild, cmd.ForceBuild, cmd.ForceDeploy, log.GetInstance())
	if err != nil {
		log.Fatalf("Error deploying dependencies: %v", err)
	}

	// Build images
	builtImages := make(map[string]string)
	if cmd.SkipBuild == false {
		builtImages, err = build.All(config, generatedConfig.GetActive(), client, cmd.SkipPush, false, cmd.ForceBuild, cmd.BuildSequential, log.GetInstance())
		if err != nil {
			if strings.Index(err.Error(), "no space left on device") != -1 {
				err = fmt.Errorf("%v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
			}

			log.Fatal(err)
		}

		// Save config if an image was built
		if len(builtImages) > 0 {
			err := generated.SaveConfig(generatedConfig)
			if err != nil {
				log.Fatalf("Error saving generated config: %v", err)
			}
		}
	}

	// What deployments should be deployed
	deployments := []string{}
	if cmd.Deployments != "" {
		deployments = strings.Split(cmd.Deployments, ",")
		for index := range deployments {
			deployments[index] = strings.TrimSpace(deployments[index])
		}
	}

	// Deploy all defined deployments
	err = deploy.All(config, generatedConfig.GetActive(), client, false, cmd.ForceDeploy, builtImages, deployments, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Save Config
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatalf("Error saving generated config: %v", err)
	}

	log.Donef("Successfully deployed!")

	log.Infof("\r         \nRun: \n- `%s` to create an ingress for the app and open it in the browser \n- `%s` to open a shell into the container \n- `%s` to show the container logs\n- `%s` to open the management ui\n- `%s` to analyze the space for potential issues\n", ansi.Color("devspace open", "white+b"), ansi.Color("devspace enter", "white+b"), ansi.Color("devspace logs", "white+b"), ansi.Color("devspace ui", "white+b"), ansi.Color("devspace analyze", "white+b"))
}

func (cmd *DeployCmd) validateFlags() {
	if cmd.SkipBuild && cmd.ForceBuild {
		log.Fatal("Flags --skip-build & --force-build cannot be used together")
	}
}
