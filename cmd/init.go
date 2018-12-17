package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/devspace/docker"
	"github.com/covexo/devspace/pkg/devspace/generator"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/dockerfile"
	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/spf13/cobra"
)

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	flags          *InitCmdFlags
	chartGenerator *generator.ChartGenerator
	defaultImage   *v1.ImageConfig
}

// InitCmdFlags are the flags available for the init-command
type InitCmdFlags struct {
	reconfigure      bool
	overwrite        bool
	skipQuestions    bool
	templateRepoURL  string
	templateRepoPath string
	language         string

	cloudProvider                     string
	useDevSpaceCloud                  bool
	addDevSpaceCloudToLocalKubernetes bool
	namespace                         string
	createInternalRegistry            bool
	registryURL                       string
	defaultImageName                  string
	createPullSecret                  bool
}

// InitCmdFlagsDefault are the default flags for InitCmdFlags
var InitCmdFlagsDefault = &InitCmdFlags{
	reconfigure:      false,
	overwrite:        false,
	skipQuestions:    false,
	templateRepoURL:  "https://github.com/covexo/devspace-templates.git",
	templateRepoPath: "",
	language:         "",

	cloudProvider:                     "",
	useDevSpaceCloud:                  false,
	addDevSpaceCloudToLocalKubernetes: false,
	namespace:                         "",
	createInternalRegistry:            false,
}

func init() {
	cmd := &InitCmd{
		flags: InitCmdFlagsDefault,
	}
	cobraCmd := &cobra.Command{
		Use:   "init",
		Short: "Initializes your DevSpace",
		Long: `
#######################################################
#################### devspace init ####################
#######################################################
Gets your project ready to start a DevSpaces.
Creates the following files and directories:

YOUR_PROJECT_PATH/
|
|-- Dockerfile
|
|-- chart/
|   |-- Chart.yaml
|   |-- values.yaml
|   |-- templates/
|       |-- deployment.yaml
|       |-- service.yaml
|       |-- ingress.yaml
|
|-- .devspace/
|   |-- .gitignore
|   |-- cluster.yaml
|   |-- config.yaml

#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().BoolVarP(&cmd.flags.reconfigure, "reconfigure", "r", cmd.flags.reconfigure, "Change existing configuration")
	cobraCmd.Flags().BoolVarP(&cmd.flags.overwrite, "overwrite", "o", cmd.flags.overwrite, "Overwrite existing chart files and Dockerfile")
	cobraCmd.Flags().BoolVarP(&cmd.flags.skipQuestions, "yes", "y", cmd.flags.skipQuestions, "Answer all questions with their default value")
	cobraCmd.Flags().StringVar(&cmd.flags.templateRepoURL, "templateRepoUrl", cmd.flags.templateRepoURL, "Git repository for chart templates")
	cobraCmd.Flags().StringVar(&cmd.flags.templateRepoPath, "templateRepoPath", cmd.flags.templateRepoPath, "Local path for cloning chart template repository (uses temp folder if not specified)")
	cobraCmd.Flags().StringVarP(&cmd.flags.language, "language", "l", cmd.flags.language, "Programming language of your project")
}

// Run executes the command logic
func (cmd *InitCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()

	var config *v1.Config

	configExists, _ := configutil.ConfigExists()
	if configExists && cmd.flags.reconfigure == false {
		config = configutil.GetConfig()
	} else {
		// Delete config & overwrite config
		os.Remove(configutil.ConfigPath)
		os.Remove(configutil.OverwriteConfigPath)
		os.Remove(generated.ConfigPath)

		// Create config
		config = configutil.InitConfig()

		// Set intial deployments
		config.DevSpace.Deployments = &[]*v1.DeploymentConfig{
			{
				Name:      configutil.String(configutil.DefaultDevspaceDeploymentName),
				Namespace: configutil.String(""),
				Helm: &v1.HelmConfig{
					ChartPath: configutil.String("./chart"),
					Overwrite: configutil.String("./chart/dev-overwrite.yaml"),
				},
			},
		}
	}

	configutil.Merge(&config, &v1.Config{
		Version: configutil.String(configutil.CurrentConfigVersion),
		Images: &map[string]*v1.ImageConfig{
			"default": &v1.ImageConfig{
				Name: configutil.String("devspace"),
			},
		},
		Registries: &map[string]*v1.RegistryConfig{
			"default": &v1.RegistryConfig{
				Auth: &v1.RegistryAuth{},
			},
			"internal": &v1.RegistryConfig{
				Auth: &v1.RegistryAuth{},
			},
		},
	}, true)

	imageMap := *config.Images
	cmd.defaultImage = imageMap["default"]

	cmd.initChartGenerator()

	createChart := cmd.flags.overwrite
	if !cmd.flags.overwrite {
		_, chartDirNotFound := os.Stat("chart")
		if chartDirNotFound == nil {
			if !cmd.flags.skipQuestions {
				createChart = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
					Question:               "Do you want to overwrite the existing files in /chart? (yes | no)",
					DefaultValue:           "no",
					ValidationRegexPattern: "^(yes)|(no)$",
				}) == "yes"
			}
		} else {
			createChart = true
		}
	}

	if createChart {
		cmd.initChartGenerator()
		cmd.determineLanguage()
		cmd.createChart()
	}

	if cmd.flags.reconfigure || !configExists {
		// Check if devspace cloud should be used
		if cmd.useCloudProvider() == false {
			cmd.configureDevSpace()
		}

		cmd.addDefaultService()
		cmd.addDefaultPorts()
		cmd.addDefaultSyncConfig()

		cmd.configureRegistry()

		err := configutil.SaveConfig()
		if err != nil {
			log.With(err).Fatalf("Config error: %s", err.Error())
		}
	}
}

func (cmd *InitCmd) initChartGenerator() {
	workdir, _ := os.Getwd()
	templateRepoPath := cmd.flags.templateRepoPath

	if len(templateRepoPath) == 0 {
		templateRepoPath, _ = ioutil.TempDir("", "")
		defer os.RemoveAll(templateRepoPath)
	}
	templateRepo := &generator.TemplateRepository{
		URL:       cmd.flags.templateRepoURL,
		LocalPath: templateRepoPath,
	}
	cmd.chartGenerator = &generator.ChartGenerator{
		TemplateRepo: templateRepo,
		Path:         workdir,
	}
}

func (cmd *InitCmd) useCloudProvider() bool {
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		log.Fatalf("Error loading cloud config: %v", err)
	}

	if len(providerConfig) > 1 {
		cloudProviderOptions := "("

		for name := range providerConfig {
			if len(cloudProviderOptions) > 1 {
				cloudProviderOptions += ", "
			}

			cloudProviderOptions += name
		}

		cloudProviderOptions += ")"
		cloudProviderSelected := cmd.flags.cloudProvider

		for _, ok := providerConfig[cloudProviderSelected]; ok == false && cloudProviderSelected != "no"; {
			cloudProviderSelected = strings.TrimSpace(*stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:     "Do you want to use a cloud provider? (no to skip) " + cloudProviderOptions,
				DefaultValue: cloud.DevSpaceCloudProviderName,
			}))

			_, ok = providerConfig[cloudProviderSelected]
		}

		if cloudProviderSelected != "no" {
			cmd.loginToCloudProvider(providerConfig, cloudProviderSelected)
			return true
		}
	} else {
		useDevSpaceCloud := cmd.flags.useDevSpaceCloud || cmd.flags.skipQuestions
		if !useDevSpaceCloud {
			useDevSpaceCloud = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want to use the DevSpace Cloud? (free ready-to-use Kubernetes) (yes | no)",
				DefaultValue:           "yes",
				ValidationRegexPattern: "^(yes)|(no)$",
			}) == "yes"
		}
		if useDevSpaceCloud {
			cmd.loginToCloudProvider(providerConfig, cloud.DevSpaceCloudProviderName)
			return true
		}
	}

	return false
}

func (cmd *InitCmd) loginToCloudProvider(providerConfig cloud.ProviderConfig, cloudProviderSelected string) {
	config := configutil.GetConfig()
	addToContext := cmd.flags.skipQuestions || cmd.flags.addDevSpaceCloudToLocalKubernetes
	if !addToContext {
		addToContext = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Do you want to add the DevSpace Cloud to the $HOME/.kube/config file? (yes | no)",
			DefaultValue:           "yes",
			ValidationRegexPattern: "^(yes)|(no)$",
		}) == "yes"
	}

	config.Cluster.CloudProvider = &cloudProviderSelected
	config.Cluster.CloudProviderDeployTarget = configutil.String(cloud.DefaultDeployTarget)

	err := cloud.Update(providerConfig, &cloud.UpdateOptions{
		UseKubeContext:    addToContext,
		SwitchKubeContext: true,
		SkipSaveConfig:    true,
	}, log.GetInstance())
	if err != nil {
		log.Fatalf("Couldn't authenticate to %s: %v", cloudProviderSelected, err)
	}

	log.Write([]byte("\n"))
}

func (cmd *InitCmd) configureDevSpace() {
	currentContext, err := kubeconfig.GetCurrentContext()
	if err != nil {
		log.Fatalf("Couldn't determine current kubernetes context: %v", err)
	}

	namespace := &cmd.flags.namespace
	if *namespace == "" {
		namespace = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which Kubernetes namespace should your application run in?",
			DefaultValue:           "default",
			ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
		})
	}

	config := configutil.GetConfig()
	config.Cluster.KubeContext = &currentContext
	config.Cluster.Namespace = namespace

	config.Tiller = &v1.TillerConfig{
		Namespace: namespace,
	}
}

func (cmd *InitCmd) addDefaultService() {
	config := configutil.GetConfig()
	config.DevSpace.Services = &[]*v1.ServiceConfig{
		{
			Name: configutil.String(configutil.DefaultDevspaceServiceName),
			LabelSelector: &map[string]*string{
				"devspace": configutil.String("default"),
			},
		},
	}
}

func (cmd *InitCmd) addDefaultPorts() {
	ports, err := dockerfile.GetPorts("Dockerfile")
	if err != nil {
		log.Warnf("Error parsing dockerfile: %v", err)
		return
	}
	if len(ports) == 0 {
		return
	}

	portMappings := []*v1.PortMapping{}
	for _, port := range ports {
		exposedPort := port

		portMappings = append(portMappings, &v1.PortMapping{
			LocalPort:  &exposedPort,
			RemotePort: &exposedPort,
		})
	}

	config := configutil.GetConfig()
	config.DevSpace.Ports = &[]*v1.PortForwardingConfig{
		{
			Service:      configutil.String(configutil.DefaultDevspaceServiceName),
			PortMappings: &portMappings,
		},
	}
}

func (cmd *InitCmd) addDefaultSyncConfig() {
	config := configutil.GetConfig()

	if config.DevSpace.Sync == nil {
		config.DevSpace.Sync = &[]*v1.SyncConfig{}
	}

	for _, syncPath := range *config.DevSpace.Sync {
		if *syncPath.LocalSubPath == "./" || *syncPath.ContainerPath == "/app" {
			return
		}
	}

	dockerignore, err := ioutil.ReadFile(".dockerignore")
	uploadExcludePaths := []string{}

	if err == nil {
		dockerignoreRules := strings.Split(string(dockerignore), "\n")

		for _, ignoreRule := range dockerignoreRules {
			if len(ignoreRule) > 0 {
				uploadExcludePaths = append(uploadExcludePaths, ignoreRule)
			}
		}
	}

	syncConfig := append(*config.DevSpace.Sync, &v1.SyncConfig{
		Service:            configutil.String(configutil.DefaultDevspaceServiceName),
		ContainerPath:      configutil.String("/app"),
		LocalSubPath:       configutil.String("./"),
		UploadExcludePaths: &uploadExcludePaths,
	})

	config.DevSpace.Sync = &syncConfig
}

func (cmd *InitCmd) configureRegistry() {
	dockerUsername := ""

	client, err := docker.NewClient(true)
	if err != nil {
		log.Fatalf("Cannot create docker client: %v", err)
	}

	useKaniko := false

	// Check if docker is installed
	for {
		_, err = client.Ping(context.Background())
		if err != nil {
			// Check if docker cli is installed
			dockerCliErr := exec.Command("docker").Run()
			if dockerCliErr == nil {
				useKaniko = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
					Question:               "Docker seems to be installed but is not running: " + err.Error() + " \nShould we build with kaniko instead?",
					DefaultValue:           "no",
					ValidationRegexPattern: "^(yes)|(no)$",
				}) == "yes"

				if useKaniko == false {
					continue
				}
			}

			// We use kaniko
			useKaniko = true

			// Set default build engine to kaniko, if no docker is installed
			cmd.defaultImage.Build = &v1.BuildConfig{
				Kaniko: &v1.KanikoConfig{
					Cache:     configutil.Bool(true),
					Namespace: configutil.String(""),
				},
			}
		}

		break
	}

	if useKaniko == false {
		log.StartWait("Checking Docker credentials")
		dockerAuthConfig, err := docker.GetAuthConfig(client, "", true)
		log.StopWait()

		if err == nil {
			dockerUsername = dockerAuthConfig.Username
		}

		// Don't push image in minikube
		if kubectl.IsMinikube() {
			cmd.defaultImage.SkipPush = configutil.Bool(true)
			return
		}
	}

	err = configure.Image(dockerUsername, cmd.flags.skipQuestions, cmd.flags.registryURL, cmd.flags.defaultImageName, cmd.flags.createPullSecret)
	if err != nil {
		log.Fatal(err)
	}
}

func (cmd *InitCmd) determineLanguage() {
	if len(cmd.flags.language) != 0 {
		if cmd.chartGenerator.IsSupportedLanguage(cmd.flags.language) {
			cmd.chartGenerator.Language = cmd.flags.language
		} else {
			log.Info("Language '" + cmd.flags.language + "' not supported yet. Please open an issue here: https://github.com/covexo/devspace/issues/new?title=Feature%20Request:%20Language%20%22" + cmd.flags.language + "%22")
		}
	}

	if len(cmd.chartGenerator.Language) == 0 {
		log.StartWait("Detecting programming language")

		detectedLang := ""
		supportedLanguages, err := cmd.chartGenerator.GetSupportedLanguages()
		if err == nil {
			detectedLang, _ = cmd.chartGenerator.GetLanguage()
		}

		if detectedLang == "" {
			detectedLang = "none"
		}
		if len(supportedLanguages) == 0 {
			supportedLanguages = []string{"none"}
		}

		log.StopWait()

		if !cmd.flags.skipQuestions {
			cmd.chartGenerator.Language = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "What is the major programming language of your project?\nSupported languages: " + strings.Join(supportedLanguages, ", "),
				DefaultValue:           detectedLang,
				ValidationRegexPattern: "^(" + strings.Join(supportedLanguages, ")|(") + ")$",
			})
		}
	}
}

func (cmd *InitCmd) createChart() {
	err := cmd.chartGenerator.CreateChart()
	if err != nil {
		log.Fatalf("Error while creating Helm chart and Dockerfile: %s", err.Error())
	}
}
