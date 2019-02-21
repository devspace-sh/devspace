package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	latest "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/devspace/docker"
	"github.com/covexo/devspace/pkg/devspace/generator"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/fsutil"
	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/ptr"
	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

const configGitignore = `logs/
overwrite.yaml
generated.yaml
`

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	flags          *InitCmdFlags
	chartGenerator *generator.ChartGenerator
	defaultImage   *latest.ImageConfig

	port      string
	imageName string
}

// InitCmdFlags are the flags available for the init-command
type InitCmdFlags struct {
	reconfigure      bool
	overwrite        bool
	useCloud         bool
	templateRepoURL  string
	templateRepoPath string
}

// InitCmdFlagsDefault are the default flags for InitCmdFlags
var InitCmdFlagsDefault = &InitCmdFlags{
	reconfigure: false,
	overwrite:   false,
	useCloud:    true,

	templateRepoURL:  "https://github.com/covexo/devspace-templates.git",
	templateRepoPath: "",
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
|   |-- generated.yaml
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
	cobraCmd.Flags().BoolVar(&cmd.flags.useCloud, "cloud", cmd.flags.useCloud, "Use the devspace.cloud to initialize project")
}

// Run executes the command logic
func (cmd *InitCmd) Run(cobraCmd *cobra.Command, args []string) {
	var config *latest.Config

	configExists := configutil.ConfigExists()
	if configExists && cmd.flags.reconfigure == false {
		log.StartFileLogging()

		config = configutil.GetBaseConfig()
	} else {
		// Delete config & overwrite config
		os.RemoveAll(".devspace")

		// Start file logging
		log.StartFileLogging()

		// Create config
		config = configutil.InitConfig()

		// Set intial deployments
		config.Deployments = &[]*latest.DeploymentConfig{
			{
				Name: ptr.String(configutil.DefaultDevspaceDeploymentName),
				Helm: &latest.HelmConfig{
					ChartPath: ptr.String("./chart"),
				},
			},
		}

		// Auto reload configuration
		config.Dev.AutoReload = &latest.AutoReloadConfig{
			Deployments: &[]*string{ptr.String(configutil.DefaultDevspaceDeploymentName)},
		}

		// Override Entrypoint
		config.Dev.OverrideImages = &[]*latest.ImageOverrideConfig{
			&latest.ImageOverrideConfig{
				Name:       ptr.String("default"),
				Entrypoint: &[]*string{ptr.String("sleep"), ptr.String("999999999999")},
			},
		}
	}

	configutil.Merge(&config, &latest.Config{
		Version: ptr.String(latest.Version),
		Images: &map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Name: ptr.String("devspace"),
			},
		},
	})

	// Print devspace logo
	log.PrintLogo()

	cmd.defaultImage = (*config.Images)["default"]
	cmd.initChartGenerator()

	createChart := cmd.flags.overwrite
	if !cmd.flags.overwrite {
		_, chartDirNotFound := os.Stat("chart")
		if chartDirNotFound == nil {
			createChart = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:     "Do you want to overwrite existing files in /chart?",
				DefaultValue: "no",
				Options:      []string{"yes", "no"},
			}) == "yes"
		} else {
			createChart = true
		}
	}

	if cmd.flags.reconfigure || !configExists {
		// Check if devspace cloud should be used
		if cmd.flags.useCloud == false {
			cmd.configureDevSpace()
		} else {
			// Configure cloud provider
			config.Cluster.CloudProvider = ptr.String(cloud.DevSpaceCloudProviderName)

			// Print target
			log.Infof("Using devspace.cloud - if you want to use your cluster run `%s`", ansi.Color("devspace init --cloud=false", "white+b"))

			// Login and login into registries if necessary
			_, err := cloud.GetCurrentProvider(log.GetInstance())
			if err != nil {
				log.Fatalf("Error login into cloud provider: %v", err)
			}
		}

		// Configure .devspace/config.yaml
		cmd.addDefaultSelector()
		cmd.addDefaultPorts()
		cmd.addDefaultSyncConfig()
		cmd.configureImage()

		// Get image name
		if len(*config.Images) > 0 {
			for _, imageConf := range *config.Images {
				cmd.imageName = *imageConf.Name
				break
			}
		}

		err := configutil.SaveBaseConfig()
		if err != nil {
			log.With(err).Fatalf("Config error: %s", err.Error())
		}

		configDir := filepath.Dir(configutil.ConfigPath)

		// Check if .gitignore exists
		_, err = os.Stat(filepath.Join(configDir, ".gitignore"))
		if os.IsNotExist(err) {
			fsutil.WriteToFile([]byte(configGitignore), filepath.Join(configDir, ".gitignore"))
		}
	}

	// Create chart and dockerfile
	if createChart {
		cmd.initChartGenerator()
		cmd.determineLanguage()
		cmd.createChart()

		cmd.replacePlaceholder()
	}

	log.Done("Project successfully initialized")

	if cmd.flags.useCloud {
		log.Infof("\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"))
	} else {
		log.Infof("Run:\n- `%s` to develop application\n- `%s` to deploy application", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
	}
}

func (cmd *InitCmd) replacePlaceholder() {
	data, err := ioutil.ReadFile("chart/values.yaml")
	if err != nil {
		log.Fatal("Couldn't find chart/values.yaml")
	}

	newContent := string(data)
	newContent = strings.Replace(newContent, "#image#", cmd.imageName, -1)
	newContent = strings.Replace(newContent, "#port#", cmd.port, -1)

	err = ioutil.WriteFile("chart/values.yaml", []byte(newContent), 0644)
	if err != nil {
		log.Fatal("Error writing chart/values.yaml")
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

func (cmd *InitCmd) configureDevSpace() {
	currentContext, err := kubeconfig.GetCurrentContext()
	if err != nil {
		log.Fatalf("Couldn't determine current kubernetes context: %v", err)
	}

	namespace := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:     "Which namespace should the app run in?",
		DefaultValue: "default",
	})

	config := configutil.GetConfig()
	config.Cluster.KubeContext = &currentContext
	config.Cluster.Namespace = namespace
}

func (cmd *InitCmd) addDefaultSelector() {
	config := configutil.GetConfig()
	config.Dev.Selectors = &[]*latest.SelectorConfig{
		{
			Name: ptr.String(configutil.DefaultDevspaceServiceName),
			LabelSelector: &map[string]*string{
				"app.kubernetes.io/name":      ptr.String("devspace-app"),
				"app.kubernetes.io/component": ptr.String("default"),
			},
		},
	}
}

func (cmd *InitCmd) addDefaultPorts() {
	port := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question: "Which port is the app listening on? (Default: 3000)",
	})
	if port == "" {
		port = "3000"
	}

	portMappings := []*latest.PortMapping{}
	exposedPort, err := strconv.Atoi(port)
	if err == nil {
		portMappings = append(portMappings, &latest.PortMapping{
			LocalPort:  &exposedPort,
			RemotePort: &exposedPort,
		})
	}

	config := configutil.GetConfig()
	config.Dev.Ports = &[]*latest.PortForwardingConfig{
		{
			Selector:     ptr.String(configutil.DefaultDevspaceServiceName),
			PortMappings: &portMappings,
		},
	}

	cmd.port = port
}

func (cmd *InitCmd) addDefaultSyncConfig() {
	config := configutil.GetConfig()

	if config.Dev.Sync == nil {
		config.Dev.Sync = &[]*latest.SyncConfig{}
	}

	for _, syncPath := range *config.Dev.Sync {
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

	syncConfig := append(*config.Dev.Sync, &latest.SyncConfig{
		Selector:           ptr.String(configutil.DefaultDevspaceServiceName),
		ContainerPath:      ptr.String("/app"),
		LocalSubPath:       ptr.String("./"),
		UploadExcludePaths: &uploadExcludePaths,
	})

	config.Dev.Sync = &syncConfig
}

func (cmd *InitCmd) configureImage() {
	dockerUsername := ""
	useKaniko := false

	// Get docker client
	client, err := docker.NewClient(true)
	if err != nil {
		log.Fatalf("Cannot create docker client: %v", err)
	}

	// Check if docker is installed
	for {
		_, err = client.Ping(context.Background())
		if err != nil {
			// Check if docker cli is installed
			err := exec.Command("docker").Run()
			if err == nil {
				if cmd.flags.useCloud {
					log.Fatal("Docker seems to be installed but is not running. Please start docker and restart `devspace init`")
				}

				useKaniko = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
					Question:               "Docker seems to be installed but is not running: " + err.Error() + " \nShould we build with kaniko instead?",
					DefaultValue:           "no",
					ValidationRegexPattern: "^(yes)|(no)$",
				}) == "yes"

				if useKaniko == false {
					continue
				}
			} else if cmd.flags.useCloud {
				log.Fatal("Please install docker in order to use `devspace init`")
			}

			// We use kaniko
			useKaniko = true

			// Set default build engine to kaniko, if no docker is installed
			cmd.defaultImage.Build = &latest.BuildConfig{
				Kaniko: &latest.KanikoConfig{
					Cache:     ptr.Bool(true),
					Namespace: ptr.String(""),
				},
			}
		}

		break
	}

	if useKaniko == false && cmd.flags.useCloud == false {
		log.StartWait("Checking Docker credentials")
		dockerAuthConfig, err := docker.GetAuthConfig(client, "", true)
		log.StopWait()

		if err == nil {
			dockerUsername = dockerAuthConfig.Username
		}

		// Don't push image in minikube
		if kubectl.IsMinikube() {
			cmd.defaultImage.SkipPush = ptr.Bool(true)
			return
		}
	}

	err = configure.Image(dockerUsername, cmd.flags.useCloud)
	if err != nil {
		log.Fatal(err)
	}
}

func (cmd *InitCmd) determineLanguage() {
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
		Question:     "Select programming language of project",
		DefaultValue: detectedLang,
		Options:      supportedLanguages,
	})
}

func (cmd *InitCmd) createChart() {
	err := cmd.chartGenerator.CreateChart()
	if err != nil {
		log.Fatalf("Error while creating Helm chart and Dockerfile: %s", err.Error())
	}
}
