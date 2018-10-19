package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/kubectl"

	"github.com/covexo/devspace/pkg/util/dockerfile"
	"github.com/covexo/devspace/pkg/util/kubeconfig"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/configure"

	"github.com/covexo/devspace/pkg/devspace/builder/docker"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/generator"
	"github.com/covexo/devspace/pkg/util/log"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/spf13/cobra"
)

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	flags          *InitCmdFlags
	workdir        string
	chartGenerator *generator.ChartGenerator
	defaultImage   *v1.ImageConfig
}

// InitCmdFlags are the flags available for the init-command
type InitCmdFlags struct {
	reconfigure      bool
	overwrite        bool
	templateRepoURL  string
	templateRepoPath string
	language         string
}

// InitCmdFlagsDefault are the default flags for InitCmdFlags
var InitCmdFlagsDefault = &InitCmdFlags{
	reconfigure:      false,
	overwrite:        false,
	templateRepoURL:  "https://github.com/covexo/devspace-templates.git",
	templateRepoPath: "",
	language:         "",
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
	cobraCmd.Flags().StringVar(&cmd.flags.templateRepoURL, "templateRepoUrl", cmd.flags.templateRepoURL, "Git repository for chart templates")
	cobraCmd.Flags().StringVar(&cmd.flags.templateRepoPath, "templateRepoPath", cmd.flags.templateRepoPath, "Local path for cloning chart template repository (uses temp folder if not specified)")
	cobraCmd.Flags().StringVarP(&cmd.flags.language, "language", "l", cmd.flags.language, "Programming language of your project")
}

// Run executes the command logic
func (cmd *InitCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()

	workdir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Unable to determine current workdir: %s", err.Error())
	}

	cmd.workdir = workdir

	var config *v1.Config

	configExists, _ := configutil.ConfigExists()
	if configExists && cmd.flags.reconfigure == false {
		config = configutil.GetConfig()
	} else {
		// Delete config & overwrite config
		os.Remove(filepath.Join(workdir, configutil.ConfigPath))
		os.Remove(filepath.Join(workdir, configutil.OverwriteConfigPath))
		os.Remove(filepath.Join(workdir, generated.ConfigPath))

		// Create config
		config = configutil.InitConfig()

		// Set intial deployments
		config.DevSpace.Deployments = &[]*v1.DeploymentConfig{
			{
				Name:      configutil.String(configutil.DefaultDevspaceDeploymentName),
				Namespace: configutil.String(""),
				Helm: &v1.HelmConfig{
					ChartPath: configutil.String("./chart"),
				},
			},
		}
	}

	configutil.Merge(config, &v1.Config{
		Version: configutil.String(configutil.CurrentConfigVersion),
		DevSpace: &v1.DevSpaceConfig{
			Deployments: &[]*v1.DeploymentConfig{},
		},
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
	})

	imageMap := *config.Images
	cmd.defaultImage = imageMap["default"]

	cmd.initChartGenerator()

	createChart := cmd.flags.overwrite

	if !cmd.flags.overwrite {
		_, chartDirNotFound := os.Stat(cmd.workdir + "/chart")
		if chartDirNotFound == nil {
			overwriteAnswer := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want to overwrite the existing files in /chart? (yes | no)",
				DefaultValue:           "no",
				ValidationRegexPattern: "^(yes)|(no)$",
			})
			createChart = (*overwriteAnswer == "yes")
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
		Path:         cmd.workdir,
	}
}

func (cmd *InitCmd) useCloudProvider() bool {
	config := configutil.GetConfig()
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		log.Fatalf("Error loading cloud config: %v", err)
	}

	if len(providerConfig) > 1 {
		cloudProvider := "("

		for name := range providerConfig {
			if len(cloudProvider) > 1 {
				cloudProvider += ", "
			}

			cloudProvider += name
		}

		cloudProvider += ")"
		cloudProviderSelected := ""

		for ok := false; ok == false && cloudProviderSelected != "no"; {
			cloudProviderSelected = strings.TrimSpace(*stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:     "Do you want to use a cloud provider? (no to skip) " + cloudProvider,
				DefaultValue: cloud.DevSpaceCloudProviderName,
			}))

			_, ok = providerConfig[cloudProviderSelected]
		}

		if cloudProviderSelected != "no" {
			addToContext := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want to add the cloud provider to the $HOME/.kube/config file? (yes | no)",
				DefaultValue:           "yes",
				ValidationRegexPattern: "^(yes)|(no)$",
			}) == "yes"

			config.Cluster.CloudProvider = &cloudProviderSelected

			log.StartWait("Logging into cloud provider " + providerConfig[cloudProviderSelected].Host + cloud.LoginEndpoint + "...")
			err := cloud.Update(providerConfig, config, addToContext, true)
			log.StopWait()
			if err != nil {
				log.Fatalf("Couldn't authenticate to devspace cloud: %v", err)
			}

			return true
		}
	} else {
		useDevSpaceCloud := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Do you want to use the devspace cloud? (free ready-to-use kubernetes) (yes | no)",
			DefaultValue:           "yes",
			ValidationRegexPattern: "^(yes)|(no)$",
		}) == "yes"

		if useDevSpaceCloud {
			addToContext := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want to add the devspace-cloud to the $HOME/.kube/config file? (yes | no)",
				DefaultValue:           "yes",
				ValidationRegexPattern: "^(yes)|(no)$",
			}) == "yes"

			config.Cluster.CloudProvider = configutil.String(cloud.DevSpaceCloudProviderName)

			log.StartWait("Logging into cloud provider " + providerConfig[cloud.DevSpaceCloudProviderName].Host + cloud.LoginEndpoint + "...")
			err := cloud.Update(providerConfig, config, addToContext, true)
			log.StopWait()
			if err != nil {
				log.Fatalf("Couldn't authenticate to devspace cloud: %v", err)
			}

			return true
		}
	}

	return false
}

func (cmd *InitCmd) configureDevSpace() {
	currentContext, err := kubeconfig.GetCurrentContext()
	if err != nil {
		log.Fatalf("Couldn't determine current kubernetes context: %v", err)
	}

	namespace := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Which Kubernetes namespace should your application run in?",
		DefaultValue:           "default",
		ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
	})

	config := configutil.GetConfig()
	config.Cluster.KubeContext = &currentContext
	config.Cluster.Namespace = namespace
}

func (cmd *InitCmd) addDefaultPorts() {
	dockerfilePath := filepath.Join(cmd.workdir, "Dockerfile")
	ports, err := dockerfile.GetPorts(dockerfilePath)
	if err != nil {
		log.Warnf("Error parsing dockerfile %s: %v", dockerfilePath, err)
		return
	}
	if len(ports) == 0 {
		return
	}

	portMappings := []*v1.PortMapping{}
	for _, port := range ports {
		portMappings = append(portMappings, &v1.PortMapping{
			LocalPort:  &port,
			RemotePort: &port,
		})
	}

	config := configutil.GetConfig()
	config.DevSpace.Ports = &[]*v1.PortForwardingConfig{
		{
			LabelSelector: &map[string]*string{
				"release": configutil.String(configutil.DefaultDevspaceDeploymentName),
			},
			PortMappings: &portMappings,
		},
	}
}

func (cmd *InitCmd) addDefaultSyncConfig() {
	config := configutil.GetConfig()

	for _, syncPath := range *config.DevSpace.Sync {
		if *syncPath.LocalSubPath == "./" || *syncPath.ContainerPath == "/app" {
			return
		}
	}
	dockerignoreFile := filepath.Join(cmd.workdir, ".dockerignore")
	dockerignore, err := ioutil.ReadFile(dockerignoreFile)
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
		ContainerPath: configutil.String("/app"),
		LocalSubPath:  configutil.String("./"),
		LabelSelector: &map[string]*string{
			"release": configutil.String(configutil.DefaultDevspaceDeploymentName),
		},
		UploadExcludePaths: &uploadExcludePaths,
	})

	config.DevSpace.Sync = &syncConfig
}

func (cmd *InitCmd) configureRegistry() {
	dockerUsername := ""
	createInternalRegistryDefaultAnswer := "yes"

	imageBuilder, err := docker.NewBuilder("", "", "", false)
	if err == nil {
		log.StartWait("Checking Docker credentials")
		dockerAuthConfig, err := imageBuilder.Authenticate("", "", true)
		log.StopWait()

		if err == nil {
			dockerUsername = dockerAuthConfig.Username
			if dockerUsername != "" {
				createInternalRegistryDefaultAnswer = "no"
			}
		}
	} else {
		// Set default build engine to kaniko, if no docker is installed
		cmd.defaultImage.Build = &v1.BuildConfig{
			Kaniko: &v1.KanikoConfig{
				Cache:     configutil.Bool(true),
				Namespace: configutil.String(""),
			},
		}
	}

	// Only deploy registry in minikube
	if kubectl.IsMinikube() {
		createInternalRegistry := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Should we create a private registry within your Kubernetes cluster for you? (yes | no)",
			DefaultValue:           createInternalRegistryDefaultAnswer,
			ValidationRegexPattern: "^(yes)|(no)$",
		})

		if *createInternalRegistry == "yes" {
			err := configure.InternalRegistry()
			if err != nil {
				log.Fatal(err)
			}

			return
		}
	}

	err = configure.ImageName(dockerUsername)
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

		cmd.chartGenerator.Language = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "What is the major programming language of your project?\nSupported languages: " + strings.Join(supportedLanguages, ", "),
			DefaultValue:           detectedLang,
			ValidationRegexPattern: "^(" + strings.Join(supportedLanguages, ")|(") + ")$",
		})
	}
}

func (cmd *InitCmd) createChart() {
	err := cmd.chartGenerator.CreateChart()
	if err != nil {
		log.Fatalf("Error while creating Helm chart and Dockerfile: %s", err.Error())
	}
}
