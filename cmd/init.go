package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/builder/helper"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

const configGitignore = "\n\n# Exclude .devspace generated files\n.devspace/\n"

const (
	// Cluster options
	useDevSpaceCloud           = "DevSpace Cloud (managed cluster)"
	useDevSpaceCloudOwnCluster = "DevSpace Cloud (connect your own cluster)"
	useCurrentContext          = "Use current kubectl context (no server-side component)"

	// Cluster connect options
	demoClusterOption    = "Use Demo cluster (managed by DevSpace Cloud)"
	connectClusterOption = "Connect cluster to DevSpace Cloud"

	// Dockerfile not found options
	useExistingDockerfileOption = "Use the Dockerfile in ./Dockerfile"
	createDockerfileOption      = "Create a Dockerfile for me"
	enterDockerfileOption       = "Enter path to your Dockerfile"
	enterManifestsOption        = "Enter path to your Kubernetes manifests"
	enterHelmChartOption        = "Enter path to your Helm chart"
	useExistingImageOption      = "Use existing image (e.g. from Docker Hub)"
)

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	// Flags
	Reconfigure bool
	Dockerfile  string
	Context     string

	providerName        *string
	useCloud            bool
	dockerfileGenerator *generator.DockerfileGenerator
}

// NewInitCmd creates a new init command
func NewInitCmd() *cobra.Command {
	cmd := &InitCmd{}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initializes DevSpace in the current folder",
		Long: `
#######################################################
#################### devspace init ####################
#######################################################
Initializes a new devspace project within the current
folder. Creates a devspace.yaml with all configuration.
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}

	initCmd.Flags().BoolVarP(&cmd.Reconfigure, "reconfigure", "r", false, "Change existing configuration")
	initCmd.Flags().StringVar(&cmd.Context, "context", "", "Context path to use for intialization")
	initCmd.Flags().StringVar(&cmd.Dockerfile, "dockerfile", helper.DefaultDockerfilePath, "Dockerfile to use for initialization")

	return initCmd
}

// Run executes the command logic
func (cmd *InitCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Check if config already exists
	configExists := configutil.ConfigExists()
	if configExists && cmd.Reconfigure == false {
		log.Info("Config already exists. If you want to recreate the config please run `devspace init --reconfigure`")
		log.Infof("\r          \nIf you want to continue with the existing config, run:\n- `%s` to develop application\n- `%s` to deploy application\n", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
		return
	}

	// Delete config & overwrite config
	os.RemoveAll(".devspace")

	// Delete configs path
	os.Remove(constants.DefaultConfigsPath)

	// Delete config & overwrite config
	os.Remove(constants.DefaultConfigPath)

	// Delete config & overwrite config
	os.Remove(constants.DefaultVarsPath)

	// Create config
	config := configutil.InitConfig()

	// Print DevSpace logo
	log.PrintLogo()

	// Check if user wants to use devspace cloud
	cmd.checkIfDevSpaceCloud()

	// Add deployment and image config
	deploymentName, err := getDeploymentName()
	if err != nil {
		log.Fatal(err)
	}

	var newImage *latest.ImageConfig
	var newDeployment *latest.DeploymentConfig
	var selectedOption string

	// Check if dockerfile exists
	addFromDockerfile := true

	_, err = os.Stat(cmd.Dockerfile)
	if err != nil {
		selectedOption = survey.Question(&survey.QuestionOptions{
			Question:     "This project does not have a Dockerfile. What do you want to do?",
			DefaultValue: createDockerfileOption,
			Options: []string{
				createDockerfileOption,
				enterDockerfileOption,
				enterManifestsOption,
				enterHelmChartOption,
				useExistingImageOption,
			},
		})
	} else {
		selectedOption = survey.Question(&survey.QuestionOptions{
			Question:     "How do you want to initialize this project?",
			DefaultValue: useExistingDockerfileOption,
			Options: []string{
				useExistingDockerfileOption,
				enterDockerfileOption,
				enterManifestsOption,
				enterHelmChartOption,
				useExistingImageOption,
			},
		})
	}

	if selectedOption == createDockerfileOption {
		// Containerize application if necessary
		err = generator.ContainerizeApplication(cmd.Dockerfile, ".", "")
		if err != nil {
			log.Fatalf("Error containerizing application: %v", err)
		}
	} else if selectedOption == enterDockerfileOption {
		cmd.Dockerfile = survey.Question(&survey.QuestionOptions{
			Question: "Please enter a path to your Dockerfile (e.g. ./MyDockerfile)",
		})
	} else if selectedOption == enterManifestsOption {
		addFromDockerfile = false
		manifests := survey.Question(&survey.QuestionOptions{
			Question: "Please enter Kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. 'manifests/**' or 'kube/pod.yaml')",
		})

		newDeployment, err = configure.GetKubectlDeployment(deploymentName, manifests)
		if err != nil {
			log.Fatal(err)
		}
	} else if selectedOption == enterHelmChartOption {
		addFromDockerfile = false
		chartName := survey.Question(&survey.QuestionOptions{
			Question: "Please enter the path to a helm chart to deploy (e.g. ./chart)",
		})

		newDeployment, err = configure.GetHelmDeployment(deploymentName, chartName, "", "")
		if err != nil {
			log.Fatal(err)
		}
	} else if selectedOption == useExistingImageOption {
		addFromDockerfile = false
		existingImageName := survey.Question(&survey.QuestionOptions{
			Question: "Please enter a docker image to deploy (e.g. gcr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)",
		})

		newImage, newDeployment, err = configure.GetImageComponentDeployment(deploymentName, existingImageName)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Check if dockerfile exists now
	if addFromDockerfile {
		_, err = os.Stat(cmd.Dockerfile)
		if err != nil {
			log.Fatalf("Couldn't find dockerfile at '%s'. Please make sure you have a Dockerfile at the specified location", cmd.Dockerfile)
		}

		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		newImage, newDeployment, err = configure.GetDockerfileComponentDeployment(config, generatedConfig, deploymentName, "", cmd.Dockerfile, cmd.Context)
		if err != nil {
			log.Fatal(err)
		}
	}

	if newImage != nil {
		(*config.Images)["default"] = newImage
	}
	if newDeployment != nil {
		config.Deployments = &[]*latest.DeploymentConfig{newDeployment}
	}

	// Add the development configuration
	cmd.addDevConfig()

	// Save config
	err = configutil.SaveLoadedConfig()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	// Check if .gitignore exists
	_, err = os.Stat(".gitignore")
	if os.IsNotExist(err) {
		fsutil.WriteToFile([]byte(configGitignore), ".gitignore")
	} else {
		gitignoreContent, err := ioutil.ReadFile(".gitignore")
		if err != nil {
			log.Warnf("Error reading .gitignore: %v", err)
		} else {
			gitignoreRegexp := regexp.MustCompile("(?ms)(^|[^!]).devspace/(\\n|$)")

			if gitignoreRegexp.MatchString(string(gitignoreContent)) == false {
				gitignore, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_WRONLY, 0600)
				if err != nil {
					log.Warnf("Error writing to .gitignore: %v", err)
				} else {
					defer gitignore.Close()

					if _, err = gitignore.WriteString(configGitignore); err != nil {
						log.Warnf("Error writing to .gitignore: %v", err)
					}
				}
			}
		}
	}

	log.Done("Project successfully initialized")

	if cmd.useCloud {
		log.Infof("\r          \nPlease run: \n- `%s` to create a new space\n- `%s` to use an existing space\n", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"))
	} else {
		log.Infof("\r          \nRun:\n- `%s` to develop application\n- `%s` to deploy application\n", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
	}
}

func (cmd *InitCmd) checkIfDevSpaceCloud() {
	cmd.useCloud = true
	connectCluster := false

	// Get provider configuration
	providerConfig, err := cloudconfig.ParseProviderConfig()
	if err != nil {
		log.Fatalf("Error loading provider config: %v", err)
	}

	// Check if kubectl exists
	if kubeconfig.ConfigExists() {
		var options []string
		if providerConfig.Default == "" || providerConfig.Default == cloudconfig.DevSpaceCloudProviderName {
			options = []string{useDevSpaceCloud, useDevSpaceCloudOwnCluster, useCurrentContext}
		} else {
			options = []string{useDevSpaceCloud, useCurrentContext}
		}

		selectedOption := survey.Question(&survey.QuestionOptions{
			Question:     "Which Kubernetes cluster do you want to use?",
			DefaultValue: useDevSpaceCloud,
			Options:      options,
		})

		if selectedOption == useDevSpaceCloud {
			cmd.useCloud = true
		} else if selectedOption == useDevSpaceCloudOwnCluster {
			cmd.useCloud = true
			connectCluster = true
		} else {
			cmd.useCloud = false
		}
	}

	// Check if DevSpace Cloud should be used
	if cmd.useCloud == false {
		cmd.configureCluster()
	} else {
		// Configure cloud provider
		cmd.providerName = ptr.String(cloudconfig.DevSpaceCloudProviderName)

		// Choose cloud provider
		if providerConfig.Default != "" {
			cmd.providerName = &providerConfig.Default
		} else if len(providerConfig.Providers) > 1 {
			options := []string{}
			for _, provider := range providerConfig.Providers {
				options = append(options, provider.Name)
			}

			cmd.providerName = ptr.String(survey.Question(&survey.QuestionOptions{
				Question: "Select a cloud provider",
				Options:  options,
			}))
		}

		// Ensure user is logged in
		err = cloud.EnsureLoggedIn(providerConfig, *cmd.providerName, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}

		// Create generated yaml if cloud
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		generatedConfig.CloudSpace = &generated.CloudSpaceConfig{
			ProviderName: *cmd.providerName,
		}

		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatal(err)
		}

		// Check if we should connect cluster
		if connectCluster {
			cmd.connectCluster()
		}
	}
}

func (cmd *InitCmd) connectCluster() {
	provider, err := cloud.GetProvider(cmd.providerName, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	clusters, err := provider.GetClusters()
	if err != nil {
		log.Fatal(err)
	}

	connectedClusters := []string{}
	for _, cluster := range clusters {
		if cluster.Owner != nil {
			connectedClusters = append(connectedClusters, cluster.Name)
		}
	}

	connectCluster := false
	if len(connectedClusters) == 0 {
		connectCluster = survey.Question(&survey.QuestionOptions{
			Question: "You do not have any clusters connected. What do you want to do?",
			Options:  []string{demoClusterOption, connectClusterOption},
		}) == connectClusterOption
	} else {
		connectedClusters = append(connectedClusters, connectClusterOption)

		connectCluster = survey.Question(&survey.QuestionOptions{
			Question: "Which cluster do you want to use?",
			Options:  connectedClusters,
		}) == connectClusterOption
	}

	// User selected connect cluster
	if connectCluster {
		err = provider.ConnectCluster(&cloud.ConnectClusterOptions{
			DeployAdmissionController: true,
			DeployIngressController:   true,
			DeployCertManager:         true,
			UseDomain:                 true,
		})
		if err != nil {
			log.Fatal(err)
		}

		log.Done("Successfully connected cluster to DevSpace Cloud")
	}
}

func getDeploymentName() (string, error) {
	absPath, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}

	dirname := filepath.Base(absPath)
	dirname = strings.ToLower(dirname)
	dirname = regexp.MustCompile("[^a-zA-Z0-9- ]+").ReplaceAllString(dirname, "")
	dirname = regexp.MustCompile("[^a-zA-Z0-9-]+").ReplaceAllString(dirname, "-")
	dirname = strings.Trim(dirname, "-")
	if len(dirname) == 0 {
		dirname = "devspace"
	}

	return dirname, nil
}

func (cmd *InitCmd) configureCluster() {
	namespace := survey.Question(&survey.QuestionOptions{
		Question:     "Which namespace should the app run in?",
		DefaultValue: "default",
	})

	config := configutil.GetConfig()
	config.Cluster.Namespace = &namespace
}

func (cmd *InitCmd) addDevConfig() {
	config := configutil.GetConfig()

	// Forward ports
	if len(*config.Deployments) > 0 && (*config.Deployments)[0].Component != nil && (*config.Deployments)[0].Component.Service != nil && (*config.Deployments)[0].Component.Service.Ports != nil && len(*(*config.Deployments)[0].Component.Service.Ports) > 0 {
		servicePort := (*(*config.Deployments)[0].Component.Service.Ports)[0]

		if servicePort.Port != nil {
			portMappings := []*latest.PortMapping{}
			exposedPort := *servicePort.Port
			portMappings = append(portMappings, &latest.PortMapping{
				LocalPort: &exposedPort,
			})

			config.Dev.Ports = &[]*latest.PortForwardingConfig{
				{
					LabelSelector: &map[string]*string{
						"app.kubernetes.io/component": (*config.Deployments)[0].Name,
					},
					PortMappings: &portMappings,
				},
			}
		}
	}

	// Specify sync path
	if len(*config.Images) > 0 && len(*config.Deployments) > 0 && (*config.Deployments)[0].Component != nil {
		if (*config.Images)["default"].Build == nil || (*config.Images)["default"].Build.Disabled == nil {
			if config.Dev.Sync == nil {
				config.Dev.Sync = &[]*latest.SyncConfig{}
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
				LabelSelector: &map[string]*string{
					"app.kubernetes.io/component": (*config.Deployments)[0].Name,
				},
				ExcludePaths: &excludePaths,
			})

			config.Dev.Sync = &syncConfig
		}
	}

	// Override image entrypoint
	if len(*config.Images) > 0 {
		config.Dev.OverrideImages = &[]*latest.ImageOverrideConfig{
			&latest.ImageOverrideConfig{
				Name:       ptr.String("default"),
				Entrypoint: &[]*string{ptr.String("sleep"), ptr.String("999999999999")},
			},
		}
	}
}
