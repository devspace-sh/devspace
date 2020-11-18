package cmd

import (
	"context"
	"github.com/devspace-cloud/devspace/cmd/restore"
	"github.com/devspace-cloud/devspace/cmd/save"
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
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
	SkipDeploy          bool
	Deployments         string
	ForceDependencies   bool
	VerboseDependencies bool

	SkipPush                bool
	AllowCyclicDependencies bool
	Dependency              []string

	RestoreVars    bool
	SaveVars       bool
	VarsSecretName string

	Wait    bool
	Timeout int

	log logpkg.Logger
}

// NewDeployCmd creates a new deploy command
func NewDeployCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &DeployCmd{
		GlobalFlags: globalFlags,
		log:         f.GetLog(),
	}

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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Print upgrade message if new version available
			upgrade.PrintUpgradeMessage()

			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}

	deployCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")
	deployCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", false, "Deploys the dependencies verbosely")

	deployCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")

	deployCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to (re-)build every image")
	deployCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	deployCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	deployCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to (re-)deploy every deployment")
	deployCmd.Flags().BoolVar(&cmd.ForceDependencies, "force-dependencies", true, "Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)")
	deployCmd.Flags().BoolVar(&cmd.SkipDeploy, "skip-deploy", false, "Skips deploying and only builds images")
	deployCmd.Flags().StringVar(&cmd.Deployments, "deployments", "", "Only deploy a specifc deployment (You can specify multiple deployments comma-separated")

	deployCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Deploys only the specific named dependencies")

	deployCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "If true will wait for pods to be running or fails after given timeout")
	deployCmd.Flags().IntVar(&cmd.Timeout, "timeout", 120, "Timeout until deploy should stop waiting")

	deployCmd.Flags().BoolVar(&cmd.RestoreVars, "restore-vars", false, "If true will restore the variables from kubernetes before loading the config")
	deployCmd.Flags().BoolVar(&cmd.SaveVars, "save-vars", false, "If true will save the variables to kubernetes after loading the config")
	deployCmd.Flags().StringVar(&cmd.VarsSecretName, "vars-secret", "devspace-vars", "The secret to restore/save the variables from/to, if --restore-vars or --save-vars is enabled")
	return deployCmd
}

// Run executes the down command logic
func (cmd *DeployCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// set config root
	cmd.log = f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(cmd.ToConfigOptions(), cmd.log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// start file logging
	logpkg.StartFileLogging()

	// validate flags
	err = cmd.validateFlags()
	if err != nil {
		return err
	}

	// load generated config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return errors.Errorf("Error loading generated.yaml: %v", err)
	}

	// use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	// create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// warn the user if we deployed into a different context before
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, true, cmd.log)
	if err != nil {
		return err
	}

	// clear the dependencies & deployments cache if necessary
	clearCache(generatedConfig, client)

	// deprecated: Fill DEVSPACE_DOMAIN vars
	err = fillDevSpaceDomainVars(client, generatedConfig)
	if err != nil {
		return err
	}

	// restore vars if wanted
	if cmd.RestoreVars {
		vars, _, err := restore.RestoreVarsFromSecret(client, cmd.VarsSecretName)
		if err != nil {
			return errors.Wrap(err, "restore vars")
		}

		generatedConfig.Vars = vars
	}

	// add current kube context to context
	config, err := configLoader.Load()
	if err != nil {
		return err
	}

	// save vars if wanted
	if cmd.SaveVars {
		err = save.SaveVarsInSecret(client, generatedConfig.Vars, cmd.VarsSecretName)
		if err != nil {
			return errors.Wrap(err, "save vars")
		}
	}

	// execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "deploy", client.CurrentContext(), client.Namespace(), config)
	if err != nil {
		return err
	}

	// create namespace if necessary
	err = client.EnsureDeployNamespaces(config, cmd.log)
	if err != nil {
		return errors.Errorf("Unable to create namespace: %v", err)
	}

	// create docker client
	dockerClient, err := f.NewDockerClient(cmd.log)
	if err != nil {
		dockerClient = nil
	}

	// create pull secrets if necessary
	err = f.NewPullSecretClient(config, client, dockerClient, cmd.log).CreatePullSecrets()
	if err != nil {
		cmd.log.Warn(err)
	}

	// create dependency manager
	manager, err := f.NewDependencyManager(config, generatedConfig, client, cmd.AllowCyclicDependencies, configOptions, cmd.log)
	if err != nil {
		return errors.Wrap(err, "new manager")
	}

	// deploy dependencies
	err = manager.DeployAll(dependency.DeployOptions{
		Dependencies:            cmd.Dependency,
		SkipPush:                cmd.SkipPush,
		ForceDeployDependencies: cmd.ForceDependencies,
		SkipBuild:               cmd.SkipBuild,
		ForceBuild:              cmd.ForceBuild,
		SkipDeploy:              cmd.SkipDeploy,
		ForceDeploy:             cmd.ForceDeploy,
		Verbose:                 cmd.VerboseDependencies,
	})
	if err != nil {
		return errors.Wrap(err, "deploy dependencies")
	}

	// only deploy if we don't want to deploy a dependency specificly
	if len(cmd.Dependency) == 0 {
		// build images
		builtImages := make(map[string]string)
		if cmd.SkipBuild == false {
			builtImages, err = f.NewBuildController(config, generatedConfig.GetActive(), client).Build(&build.Options{
				SkipPush:     cmd.SkipPush,
				ForceRebuild: cmd.ForceBuild,
				Sequential:   cmd.BuildSequential,
			}, cmd.log)
			if err != nil {
				if strings.Index(err.Error(), "no space left on device") != -1 {
					err = errors.Errorf("%v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
				}

				return err
			}

			// save cache if an image was built
			if len(builtImages) > 0 {
				err := configLoader.SaveGenerated()
				if err != nil {
					return errors.Errorf("Error saving generated config: %v", err)
				}
			}
		}

		// what deployments should be deployed
		deployments := []string{}
		if cmd.SkipDeploy == false {
			if cmd.Deployments != "" {
				deployments = strings.Split(cmd.Deployments, ",")
				for index := range deployments {
					deployments[index] = strings.TrimSpace(deployments[index])
				}
			}

			// deploy all defined deployments
			err = f.NewDeployController(config, generatedConfig.GetActive(), client).Deploy(&deploy.Options{
				ForceDeploy: cmd.ForceDeploy,
				BuiltImages: builtImages,
				Deployments: deployments,
			}, cmd.log)
			if err != nil {
				return err
			}
		}
	}

	// update last used kube context & save generated yaml
	err = updateLastKubeContext(configLoader, client, generatedConfig)
	if err != nil {
		return errors.Wrap(err, "update last kube context")
	}

	// wait if necessary
	if cmd.Wait {
		report, err := f.NewAnalyzer(client, f.GetLog()).CreateReport(client.Namespace(), analyze.Options{Wait: true, Patient: true, Timeout: cmd.Timeout})
		if err != nil {
			return errors.Wrap(err, "analyze")
		}

		if len(report) > 0 {
			return errors.Errorf(analyze.ReportToString(report))
		}
	}

	cmd.log.Donef("Successfully deployed!")
	cmd.log.Infof("\r         \nRun: \n- `%s` to create an ingress for the app and open it in the browser \n- `%s` to open a shell into the container \n- `%s` to show the container logs\n- `%s` to analyze the space for potential issues\n", ansi.Color("devspace open", "white+b"), ansi.Color("devspace enter", "white+b"), ansi.Color("devspace logs", "white+b"), ansi.Color("devspace analyze", "white+b"))
	return nil
}

func (cmd *DeployCmd) validateFlags() error {
	if cmd.SkipBuild && cmd.ForceBuild {
		return errors.New("Flags --skip-build & --force-build cannot be used together")
	}

	return nil
}

func fillDevSpaceDomainVars(client kubectl.Client, generatedConfig *generated.Config) error {
	namespace, err := client.KubeClient().CoreV1().Namespaces().Get(context.TODO(), client.Namespace(), metav1.GetOptions{})
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

func clearCache(generatedConfig *generated.Config, client kubectl.Client) {
	if generatedConfig.GetActive().LastContext != nil {
		if (generatedConfig.GetActive().LastContext.Context != "" && generatedConfig.GetActive().LastContext.Context != client.CurrentContext()) || (generatedConfig.GetActive().LastContext.Namespace != "" && generatedConfig.GetActive().LastContext.Namespace != client.Namespace()) {
			generatedConfig.GetActive().Deployments = map[string]*generated.DeploymentCache{}
			generatedConfig.GetActive().Dependencies = map[string]string{}
		}
	}
}
