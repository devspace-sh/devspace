package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/builder/helper"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const gitIgnoreFile = ".gitignore"
const dockerIgnoreFile = ".dockerignore"
const devspaceFolderGitignore = "\n\n# Ignore DevSpace cache and log folder\n.devspace/\n"
const configDockerignore = "\n\n# Ignore devspace.yaml file to prevent image rebuilding after config changes\ndevspace.yaml\n"

const (
	// Dockerfile not found options
	useExistingDockerfileOption = "Use the Dockerfile in ./Dockerfile"
	createDockerfileOption      = "Create a Dockerfile for me"
	enterDockerfileOption       = "Enter path to your Dockerfile"
	enterManifestsOption        = "Enter path to your Kubernetes manifests"
	enterHelmChartOption        = "Enter path to your Helm chart"
	useExistingImageOption      = "Use existing image (e.g. from Docker Hub)"

	// The default image name in the config
	defaultImageName = "default"
)

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	// Flags
	Reconfigure bool
	Dockerfile  string
	Context     string
	Provider    string

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
		RunE: cmd.Run,
	}

	initCmd.Flags().BoolVarP(&cmd.Reconfigure, "reconfigure", "r", false, "Change existing configuration")
	initCmd.Flags().StringVar(&cmd.Context, "context", "", "Context path to use for intialization")
	initCmd.Flags().StringVar(&cmd.Dockerfile, "dockerfile", helper.DefaultDockerfilePath, "Dockerfile to use for initialization")
	initCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	return initCmd
}

// Run executes the command logic
func (cmd *InitCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Check if config already exists
	configExists := configutil.ConfigExists()
	if configExists && cmd.Reconfigure == false {
		log.Info("Config already exists. If you want to recreate the config please run `devspace init --reconfigure`")
		log.Infof("\r         \nIf you want to continue with the existing config, run:\n- `%s` to develop application\n- `%s` to deploy application\n", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
		return nil
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

	// Add deployment and image config
	deploymentName, err := getDeploymentName()
	if err != nil {
		return err
	}

	var (
		newImage       *latest.ImageConfig
		newDeployment  *latest.DeploymentConfig
		selectedOption string
	)

	// Check if dockerfile exists
	addFromDockerfile := true

	_, err = os.Stat(cmd.Dockerfile)
	if err != nil {
		selectedOption, err = survey.Question(&survey.QuestionOptions{
			Question:     "This project does not have a Dockerfile. What do you want to do?",
			DefaultValue: createDockerfileOption,
			Options: []string{
				createDockerfileOption,
				enterDockerfileOption,
				enterManifestsOption,
				enterHelmChartOption,
				useExistingImageOption,
			},
		}, log.GetInstance())
		if err != nil {
			return err
		}
	} else {
		selectedOption, err = survey.Question(&survey.QuestionOptions{
			Question:     "How do you want to initialize this project?",
			DefaultValue: useExistingDockerfileOption,
			Options: []string{
				useExistingDockerfileOption,
				enterDockerfileOption,
				enterManifestsOption,
				enterHelmChartOption,
				useExistingImageOption,
			},
		}, log.GetInstance())
		if err != nil {
			return err
		}
	}

	if selectedOption == createDockerfileOption {
		// Containerize application if necessary
		err = generator.ContainerizeApplication(cmd.Dockerfile, ".", "", log.GetInstance())
		if err != nil {
			return errors.Wrap(err, "containerize application")
		}
	} else if selectedOption == enterDockerfileOption {
		cmd.Dockerfile, err = survey.Question(&survey.QuestionOptions{
			Question: "Please enter a path to your Dockerfile (e.g. ./MyDockerfile)",
		}, log.GetInstance())
		if err != nil {
			return err
		}
	} else if selectedOption == enterManifestsOption {
		addFromDockerfile = false
		manifests, err := survey.Question(&survey.QuestionOptions{
			Question: "Please enter Kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. 'manifests/**' or 'kube/pod.yaml')",
		}, log.GetInstance())
		if err != nil {
			return err
		}

		newDeployment, err = configure.GetKubectlDeployment(deploymentName, manifests)
		if err != nil {
			return err
		}
	} else if selectedOption == enterHelmChartOption {
		addFromDockerfile = false
		chartName, err := survey.Question(&survey.QuestionOptions{
			Question: "Please enter the path to a helm chart to deploy (e.g. ./chart)",
		}, log.GetInstance())
		if err != nil {
			return err
		}

		newDeployment, err = configure.GetHelmDeployment(deploymentName, chartName, "", "")
		if err != nil {
			return err
		}
	} else if selectedOption == useExistingImageOption {
		addFromDockerfile = false
		existingImageName, err := survey.Question(&survey.QuestionOptions{
			Question: "Please enter a docker image to deploy (e.g. gcr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)",
		}, log.GetInstance())
		if err != nil {
			return err
		}

		newImage, newDeployment, err = configure.GetImageComponentDeployment(deploymentName, existingImageName, log.GetInstance())
		if err != nil {
			return err
		}
	}

	// Check if dockerfile exists now
	if addFromDockerfile {
		_, err = os.Stat(cmd.Dockerfile)
		if err != nil {
			return errors.Errorf("Couldn't find dockerfile at '%s'. Please make sure you have a Dockerfile at the specified location", cmd.Dockerfile)
		}

		generatedConfig, err := generated.LoadConfig("")
		if err != nil {
			return err
		}

		newImage, newDeployment, err = configure.GetDockerfileComponentDeployment(config, generatedConfig, deploymentName, "", cmd.Dockerfile, cmd.Context, log.GetInstance())
		if err != nil {
			return err
		}

		// Add devspace.yaml to .dockerignore
		err = appendToIgnoreFile(dockerIgnoreFile, configDockerignore)
		if err != nil {
			log.Warn(err)
		}
	}

	// Add .devspace/ to .gitignore
	err = appendToIgnoreFile(gitIgnoreFile, devspaceFolderGitignore)
	if err != nil {
		log.Warn(err)
	}

	if newImage != nil {
		config.Images[defaultImageName] = newImage
	}
	if newDeployment != nil {
		config.Deployments = []*latest.DeploymentConfig{newDeployment}
	}

	// Add the development configuration
	err = cmd.addDevConfig()
	if err != nil {
		return err
	}

	// Save config
	err = configutil.SaveLoadedConfig()
	if err != nil {
		return err
	}

	log.WriteString("\n")
	log.Done("Project successfully initialized")
	log.Infof("\r         \nPlease run: \n- `%s` to tell DevSpace to deploy to this namespace \n- `%s` to create a new space in DevSpace Cloud\n- `%s` to use an existing space\n", ansi.Color("devspace use namespace [NAME]", "white+b"), ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"))
	return nil
}

func appendToIgnoreFile(ignoreFile, content string) error {
	// Check if ignoreFile exists
	_, err := os.Stat(ignoreFile)
	if os.IsNotExist(err) {
		fsutil.WriteToFile([]byte(content), ignoreFile)
	} else {
		fileContent, err := ioutil.ReadFile(ignoreFile)
		if err != nil {
			return errors.Errorf("Error reading file %s: %v", ignoreFile, err)
		}

		// append only if not found in file content
		if strings.Contains(string(fileContent), content) == false {
			file, err := os.OpenFile(ignoreFile, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return errors.Errorf("Error writing file %s: %v", ignoreFile, err)
			}

			defer file.Close()
			if _, err = file.WriteString(content); err != nil {
				return errors.Errorf("Error writing file %s: %v", ignoreFile, err)
			}
		}
	}
	return nil
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

	if cloud.SpaceNameValidationRegEx.MatchString(dirname) == false || len(dirname) > 42 {
		dirname = "devspace"
	}

	return dirname, nil
}

func (cmd *InitCmd) addDevConfig() error {
	config, err := configutil.GetConfig(nil)
	if err != nil {
		return err
	}

	// Forward ports
	if len(config.Images) > 0 && len(config.Deployments) > 0 && config.Deployments[0].Helm != nil && config.Deployments[0].Helm.ComponentChart != nil && *config.Deployments[0].Helm.ComponentChart == true {
		componentValues := latest.ComponentConfig{}
		err = util.Convert(config.Deployments[0].Helm.Values, &componentValues)
		if err == nil && componentValues.Service != nil && componentValues.Service.Ports != nil && len(componentValues.Service.Ports) > 0 {
			servicePort := componentValues.Service.Ports[0]
			if servicePort.Port != nil {
				localPortPtr := servicePort.Port
				var remotePortPtr *int

				if *localPortPtr < 1024 {
					log.WriteString("\n")
					log.Warn("Your application listens on a system port [0-1024]. Choose a forwarding-port to access your application via localhost.")

					portString, err := survey.Question(&survey.QuestionOptions{
						Question:     "Which forwarding port [1024-49151] do you want to use to access your application?",
						DefaultValue: strconv.Itoa(*localPortPtr + 8000),
					}, log.GetInstance())
					if err != nil {
						return err
					}

					remotePortPtr = localPortPtr

					localPort, err := strconv.Atoi(portString)
					if err != nil {
						return errors.Errorf("Error parsing port '%s'", portString)
					}
					localPortPtr = &localPort
				}
				portMappings := []*latest.PortMapping{}
				portMappings = append(portMappings, &latest.PortMapping{
					LocalPort:  localPortPtr,
					RemotePort: remotePortPtr,
				})

				// Add dev.ports config
				config.Dev.Ports = []*latest.PortForwardingConfig{
					{
						ImageName:    defaultImageName,
						PortMappings: portMappings,
					},
				}

				// Add dev.open config
				config.Dev.Open = []*latest.OpenConfig{
					&latest.OpenConfig{
						URL: "http://localhost:" + strconv.Itoa(*localPortPtr),
					},
				}
			}
		}
	}

	// Specify sync path
	if len(config.Images) > 0 {
		if (config.Images)[defaultImageName].Build == nil || (config.Images)[defaultImageName].Build.Disabled == nil {
			if config.Dev.Sync == nil {
				config.Dev.Sync = []*latest.SyncConfig{}
			}

			dockerignore, err := ioutil.ReadFile(".dockerignore")
			excludePaths := []string{}
			if err == nil {
				dockerignoreRules := strings.Split(string(dockerignore), "\n")
				for _, ignoreRule := range dockerignoreRules {
					ignoreRule = strings.TrimSpace(ignoreRule)
					if len(ignoreRule) > 0 && ignoreRule[0] != "#"[0] {
						excludePaths = append(excludePaths, ignoreRule)
					}
				}
			}

			syncConfig := append(config.Dev.Sync, &latest.SyncConfig{
				ImageName:    defaultImageName,
				ExcludePaths: excludePaths,
			})

			config.Dev.Sync = syncConfig
		}
	}

	return nil
}
