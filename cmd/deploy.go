package cmd

import (
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/cmd/flags"
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
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeployCmd holds the required data for the down cmd
type DeployCmd struct {
	*flags.GlobalFlags

	ForceBuild          bool
	SkipBuild           bool
	BuildSequential     bool
	ForceDeploy         bool
	Deployments         string
	ForceDependencies   bool
	VerboseDependencies bool

	SkipPush                bool
	AllowCyclicDependencies bool
}

// NewDeployCmd creates a new deploy command
func NewDeployCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeployCmd{GlobalFlags: globalFlags}

	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the project",
		Long: `
#######################################################
################## devspace deploy ####################
#######################################################
Deploys the current project to a Space or namespace:

devspace deploy
devspace deploy -n some-namespace
devspace deploy --kube-context=deploy-context
#######################################################`,
		Args: cobra.NoArgs,
		RunE: cmd.Run,
	}

	deployCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")
	deployCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", false, "Deploys the dependencies verbosely")

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
func (cmd *DeployCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot(log.GetInstance())
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Start file logging
	log.StartFileLogging()

	// Validate flags
	err = cmd.validateFlags()
	if err != nil {
		return err
	}

	// Load generated config
	generatedConfig, err := generated.LoadConfig(cmd.Profile)
	if err != nil {
		return errors.Errorf("Error loading generated.yaml: %v", err)
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, log.GetInstance())
	if err != nil {
		return err
	}

	// Create kubectl client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// Warn the user if we deployed into a different context before
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, true, log.GetInstance())
	if err != nil {
		return err
	}

	// Deprecated: Fill DEVSPACE_DOMAIN vars
	err = fillDevSpaceDomainVars(client, generatedConfig)
	if err != nil {
		return err
	}

	// Add current kube context to context
	configOptions := cmd.ToConfigOptions()
	config, err := configutil.GetConfig(configOptions)
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		return err
	}

	// Create namespace if necessary
	err = client.EnsureDefaultNamespace(log.GetInstance())
	if err != nil {
		return errors.Errorf("Unable to create namespace: %v", err)
	}

	// Create docker client
	dockerClient, err := docker.NewClient(log.GetInstance())
	if err != nil {
		dockerClient = nil
	}

	// Create pull secrets and private registry if necessary
	registryClient := registry.NewClient(config, client, dockerClient, log.GetInstance())
	err = registryClient.CreatePullSecrets()
	if err != nil {
		return err
	}

	// Create Dependencymanager
	manager, err := dependency.NewManager(config, generatedConfig, client, cmd.AllowCyclicDependencies, configOptions, log.GetInstance())
	if err != nil {
		return errors.Wrap(err, "new manager")
	}

	// Dependencies
	err = manager.DeployAll(dependency.DeployOptions{
		SkipPush:                cmd.SkipPush,
		ForceDeployDependencies: cmd.ForceDependencies,
		SkipBuild:               cmd.SkipBuild,
		ForceBuild:              cmd.ForceBuild,
		ForceDeploy:             cmd.ForceDeploy,
		Verbose:                 cmd.VerboseDependencies,
	})
	if err != nil {
		return errors.Wrap(err, "deploy dependencies")
	}

	// Build images
	builtImages := make(map[string]string)
	if cmd.SkipBuild == false {
		builtImages, err = build.All(config, generatedConfig.GetActive(), client, cmd.SkipPush, false, cmd.ForceBuild, cmd.BuildSequential, false, log.GetInstance())
		if err != nil {
			if strings.Index(err.Error(), "no space left on device") != -1 {
				err = errors.Errorf("%v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
			}

			return err
		}

		// Save config if an image was built
		if len(builtImages) > 0 {
			err := generated.SaveConfig(generatedConfig)
			if err != nil {
				return errors.Errorf("Error saving generated config: %v", err)
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
		return err
	}

	// Update last used kube context & save generated yaml
	err = client.UpdateLastKubeContext(generatedConfig)
	if err != nil {
		return errors.Wrap(err, "update last kube context")
	}

	log.Donef("Successfully deployed!")
	log.Infof("\r         \nRun: \n- `%s` to create an ingress for the app and open it in the browser \n- `%s` to open a shell into the container \n- `%s` to show the container logs\n- `%s` to analyze the space for potential issues\n", ansi.Color("devspace open", "white+b"), ansi.Color("devspace enter", "white+b"), ansi.Color("devspace logs", "white+b"), ansi.Color("devspace analyze", "white+b"))
	return nil
}

func (cmd *DeployCmd) validateFlags() error {
	if cmd.SkipBuild && cmd.ForceBuild {
		return errors.New("Flags --skip-build & --force-build cannot be used together")
	}

	return nil
}

func fillDevSpaceDomainVars(client kubectl.Client, generatedConfig *generated.Config) error {
	namespace, err := client.KubeClient().CoreV1().Namespaces().Get(client.Namespace(), metav1.GetOptions{})
	if err != nil {
		return nil
	}

	// Check if domain there is a domain for the space
	if namespace.Annotations == nil || namespace.Annotations[allowedIngressHostsAnnotation] == "" {
		return nil
	}

	// Remove old vars
	for varName := range generatedConfig.Vars {
		if strings.HasPrefix(varName, "DEVSPACE_SPACE_DOMAIN") {
			delete(generatedConfig.Vars, varName)
		}
	}

	// Select domain
	domains := strings.Split(namespace.Annotations[allowedIngressHostsAnnotation], ",")
	for idx, domain := range domains {
		domain = strings.Replace(domain, "*.", "", -1)
		domain = strings.Replace(domain, "*", "", -1)
		domain = strings.TrimSpace(domain)

		generatedConfig.Vars["DEVSPACE_SPACE_DOMAIN"+strconv.Itoa(idx+1)] = domain
	}

	return nil
}
