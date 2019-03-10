package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/devspace-cloud/devspace/pkg/devspace/chart"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

const configGitignore = `logs/
generated.yaml
`

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	flags               *InitCmdFlags
	dockerfileGenerator *generator.DockerfileGenerator
	defaultImage        *latest.ImageConfig

	port      string
	imageName string
}

// InitCmdFlags are the flags available for the init-command
type InitCmdFlags struct {
	reconfigure     bool
	useCloud        bool
	templateRepoURL string
}

// InitCmdFlagsDefault are the default flags for InitCmdFlags
var InitCmdFlagsDefault = &InitCmdFlags{
	reconfigure: false,
	useCloud:    true,

	templateRepoURL: "",
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
	cobraCmd.Flags().StringVar(&cmd.flags.templateRepoURL, "templateRepoUrl", cmd.flags.templateRepoURL, "Git repository for chart templates")
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

		// Init config
		cmd.initConfig(config)
	}

	// Print DevSpace logo
	log.PrintLogo()

	cmd.defaultImage = (*config.Images)["default"]

	// Containerize application if necessary
	err := generator.ContainerizeApplication(".", cmd.flags.templateRepoURL)
	if err != nil {
		log.Fatalf("Error containerizing application: %v", err)
	}

	// Create chart if necessary
	_, err = os.Stat("chart")
	if err != nil {
		chartGenerator, err := generator.NewChartGenerator(".")
		if err != nil {
			log.Fatalf("Error intializing chart generator: %v", err)
		}

		err = chartGenerator.Update(false)
		if err != nil {
			log.Fatalf("Error creating chart: %v", err)
		}
	} else {
		log.Info("Devspace detected that you already have a chart at ./chart. If you want to update the chart run `devspace update chart`")
	}

	// Create .devspace config if necessary
	if cmd.flags.reconfigure || !configExists {
		// Add default ports
		cmd.addDefaultPorts()

		// Check if kubectl exists
		if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
			cmd.flags.useCloud = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:     "Do you want to use DevSpace.cloud?",
				DefaultValue: "yes",
				Options:      []string{"yes", "no"},
			}) == "yes"
		}

		var providerName *string

		// Check if DevSpace.cloud should be used
		if cmd.flags.useCloud == false {
			cmd.configureDevSpace()
		} else {
			// Get provider configuration
			providerConfig, err := cloud.ParseCloudConfig()
			if err != nil {
				log.Fatalf("Error loading provider config: %v", err)
			}

			// Configure cloud provider
			providerName = ptr.String(cloud.DevSpaceCloudProviderName)

			// Choose cloud provider
			if len(providerConfig) > 1 {
				options := []string{}
				for providerHost := range providerConfig {
					options = append(options, providerHost)
				}

				providerName = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
					Question: "Select cloud provider",
					Options:  options,
				})
			}

			// Ensure user is logged in
			err = cloud.EnsureLoggedIn(providerConfig, *providerName, log.GetInstance())
			if err != nil {
				log.Fatal(err)
			}
		}

		// Configure .devspace/config.yaml
		cmd.addDefaultSelector()
		cmd.addDefaultSyncConfig()
		cmd.configureImage(providerName)

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

		// Create generated yaml if cloud
		if cmd.flags.useCloud {
			generatedConfig, err := generated.LoadConfig()
			if err != nil {
				log.Fatal(err)
			}

			generatedConfig.CloudSpace = &generated.CloudSpaceConfig{
				ProviderName: *providerName,
			}

			err = generated.SaveConfig(generatedConfig)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	cmd.replacePlaceholder()
	log.Done("Project successfully initialized")

	if cmd.flags.useCloud {
		log.Infof("\nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"))
	} else {
		log.Infof("Run:\n- `%s` to develop application\n- `%s` to deploy application", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
	}
}

func (cmd *InitCmd) initConfig(config *latest.Config) {
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

	// Set images
	config.Images = &map[string]*latest.ImageConfig{
		"default": &latest.ImageConfig{
			Image: ptr.String("devspace"),
		},
	}
}

func (cmd *InitCmd) replacePlaceholder() {
	config := configutil.GetConfig()

	// Get image name
	if config.Images != nil && len(*config.Images) > 0 {
		for _, imageConf := range *config.Images {
			cmd.imageName = *imageConf.Image
			break
		}

		if cmd.imageName != "" {
			err := chart.ReplacePort("chart/values.yaml", cmd.imageName)
			if err != nil {
				log.Fatalf("Couldn't replace image: %v", err)
			}
		}
	}

	// Get image port
	if cmd.port == "" {
		if config.Dev != nil && config.Dev.Ports != nil && len(*config.Dev.Ports) > 0 && (*config.Dev.Ports)[0].PortMappings != nil && len(*(*config.Dev.Ports)[0].PortMappings) > 0 {
			cmd.port = strconv.Itoa(*(*(*config.Dev.Ports)[0].PortMappings)[0].RemotePort)
		}

		if cmd.port != "" {
			err := chart.ReplacePort("chart/values.yaml", cmd.port)
			if err != nil {
				log.Fatalf("Couldn't replace port: %v", err)
			}
		}
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
	excludePaths := []string{}

	if err == nil {
		dockerignoreRules := strings.Split(string(dockerignore), "\n")

		for _, ignoreRule := range dockerignoreRules {
			if len(ignoreRule) > 0 {
				excludePaths = append(excludePaths, ignoreRule)
			}
		}
	}

	syncConfig := append(*config.Dev.Sync, &latest.SyncConfig{
		Selector:      ptr.String(configutil.DefaultDevspaceServiceName),
		ContainerPath: ptr.String("/app"),
		LocalSubPath:  ptr.String("./"),
		ExcludePaths:  &excludePaths,
	})

	config.Dev.Sync = &syncConfig
}

func (cmd *InitCmd) configureImage(providerName *string) {
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

	if useKaniko == false {
		log.StartWait("Checking Docker credentials")
		dockerAuthConfig, err := docker.GetAuthConfig(client, "", true)
		log.StopWait()

		if err == nil {
			dockerUsername = dockerAuthConfig.Username
		}

		// Don't push image in minikube
		if cmd.flags.useCloud == false && kubectl.IsMinikube() {
			cmd.defaultImage.SkipPush = ptr.Bool(true)
			return
		}
	}

	err = configure.Image(dockerUsername, providerName)
	if err != nil {
		log.Fatal(err)
	}
}
