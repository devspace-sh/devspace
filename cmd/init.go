package cmd

import (
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	latest "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	"github.com/loft-sh/devspace/pkg/devspace/generator"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SpaceNameValidationRegEx is the sapace name validation regex
var SpaceNameValidationRegEx = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$")

var gitFolderIgnoreRegex = regexp.MustCompile("/?\\.git/?")

const gitIgnoreFile = ".gitignore"
const dockerIgnoreFile = ".dockerignore"
const devspaceFolderGitignore = "\n\n# Ignore DevSpace cache and log folder\n.devspace/\n"
const configDockerignore = "\n\n# Ignore devspace.yaml file to prevent image rebuilding after config changes\ndevspace.yaml\n"

const (
	// Dockerfile not found options
	UseExistingDockerfileOption = "Use the Dockerfile in ./Dockerfile"
	CreateDockerfileOption      = "Create a Dockerfile for this project"
	EnterDockerfileOption       = "Enter path to a different Dockerfile"
	ComponentChartOption        = "Deploy with the component-chart (https://devspace.sh/component-chart/docs)"
	ManifestsOption             = "Deploy with existing Kubernetes manifests (e.g. ./kube/deployment.yaml)"
	LocalHelmChartOption        = "Deploy with a local Helm chart (e.g. ./chart/)"
	RemoteHelmChartOption       = "Deploy with a remote Helm chart"

	// The default image name in the config
	defaultImageName = "app"

	// The default name for the production profile
	productionProfileName = "production"

	// The default name for the interactive profile
	interactiveProfileName = "interactive"
)

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	// Flags
	Reconfigure bool
	Dockerfile  string
	Context     string
	Provider    string

	dockerfileGenerator *generator.DockerfileGenerator
	log                 log.Logger
}

// NewInitCmd creates a new init command
func NewInitCmd(f factory.Factory, plugins []plugin.Metadata) *cobra.Command {
	cmd := &InitCmd{
		log: f.GetLog(),
	}

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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}

	initCmd.Flags().BoolVarP(&cmd.Reconfigure, "reconfigure", "r", false, "Change existing configuration")
	initCmd.Flags().StringVar(&cmd.Context, "context", "", "Context path to use for intialization")
	initCmd.Flags().StringVar(&cmd.Dockerfile, "dockerfile", helper.DefaultDockerfilePath, "Dockerfile to use for initialization")
	initCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	return initCmd
}

// Run executes the command logic
func (cmd *InitCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Check if config already exists
	cmd.log = f.GetLog()
	configLoader := f.NewConfigLoader("")
	configExists := configLoader.Exists()
	if configExists && cmd.Reconfigure == false {
		cmd.log.Info("Config already exists. If you want to recreate the config please run `devspace init --reconfigure`")
		cmd.log.Infof("\r         \nIf you want to continue with the existing config, run:\n- `%s` to develop application\n- `%s` to deploy application\n", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
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

	// Execute plugin hook
	err := plugin.ExecutePluginHook(plugins, cobraCmd, args, "init", "", "", nil)
	if err != nil {
		return err
	}

	// Create config
	config := latest.New().(*latest.Config)

	// Create ConfigureManager
	configureManager := f.NewConfigureManager(config, cmd.log)

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

	_, err = os.Stat(cmd.Dockerfile)
	if err != nil {
		selectedOption, err = cmd.log.Question(&survey.QuestionOptions{
			Question:     "This project does not seem to have a Dockerfile. What do you want to do?",
			DefaultValue: CreateDockerfileOption,
			Options: []string{
				CreateDockerfileOption,
				EnterDockerfileOption,
			},
		})
		if err != nil {
			return err
		}
	} else {
		selectedOption, err = cmd.log.Question(&survey.QuestionOptions{
			Question:     "There is a Dockerfile in this project. Do you want to use it?",
			DefaultValue: UseExistingDockerfileOption,
			Options: []string{
				UseExistingDockerfileOption,
				EnterDockerfileOption,
			},
		})
		if err != nil {
			return err
		}
	}

	if selectedOption == CreateDockerfileOption {
		// Containerize application if necessary
		err = generator.ContainerizeApplication(cmd.Dockerfile, ".", "", cmd.log)
		if err != nil {
			return errors.Wrap(err, "containerize application")
		}
	} else if selectedOption == EnterDockerfileOption {
		cmd.Dockerfile, err = cmd.log.Question(&survey.QuestionOptions{
			Question: "Please enter a path to your Dockerfile (e.g. ./backend/Dockerfile)",
		})
		if err != nil {
			return err
		}
	}

	// Check if dockerfile exists now
	_, err = os.Stat(cmd.Dockerfile)
	if err != nil {
		return errors.Errorf("Couldn't find dockerfile at '%s'. Please make sure you have a Dockerfile at the specified location", cmd.Dockerfile)
	}

	newImage, newDeployment, err = configureManager.NewDockerfileComponentDeployment(deploymentName, "", cmd.Dockerfile, cmd.Context)
	if err != nil {
		return err
	}

	// Add devspace.yaml to .dockerignore
	err = appendToIgnoreFile(dockerIgnoreFile, configDockerignore)
	if err != nil {
		cmd.log.Warn(err)
	}

	selectedOption, err = cmd.log.Question(&survey.QuestionOptions{
		Question:     "How do you want to deploy this project?",
		DefaultValue: ComponentChartOption,
		Options: []string{
			ComponentChartOption,
			ManifestsOption,
			LocalHelmChartOption,
			RemoteHelmChartOption,
		},
	})
	if err != nil {
		return err
	}

	if selectedOption == ComponentChartOption {
		// Nothing to do
	} else if selectedOption == ManifestsOption {
		manifests, err := cmd.log.Question(&survey.QuestionOptions{
			Question: "Please enter the paths to your Kubernetes manifests (comma separated, glob patterns are allowed, e.g. 'manifests/**' or 'kube/pod.yaml')",
		})
		if err != nil {
			return err
		}

		newDeployment, err = configureManager.NewKubectlDeployment(deploymentName, manifests)
		if err != nil {
			return err
		}
	} else if selectedOption == LocalHelmChartOption || selectedOption == RemoteHelmChartOption {
		question := "Please enter the path to your Helm chart (e.g. ./chart)"
		if selectedOption == RemoteHelmChartOption {
			question = "Please enter the URL  your Helm chart (e.g. https://company.tld/mychart.tgz)"
		}

		chartName, err := cmd.log.Question(&survey.QuestionOptions{
			Question: question,
		})
		if err != nil {
			return err
		}

		newDeployment, err = configureManager.NewHelmDeployment(deploymentName, chartName, "", "")
		if err != nil {
			return err
		}
	}

	// Add .devspace/ to .gitignore
	err = appendToIgnoreFile(gitIgnoreFile, devspaceFolderGitignore)
	if err != nil {
		cmd.log.Warn(err)
	}

	if newImage != nil {
		config.Images[defaultImageName] = newImage
	}

	if newDeployment != nil {
		config.Deployments = []*latest.DeploymentConfig{newDeployment}
	}

	// Add the development configuration
	err = cmd.addDevConfig(config)
	if err != nil {
		return err
	}

	// Add the profile configuration
	err = cmd.addProfileConfig(config)
	if err != nil {
		return err
	}

	// Save config
	err = configLoader.Save(config)
	if err != nil {
		return err
	}

	cmd.log.WriteString("\n")
	cmd.log.Done("Project successfully initialized")
	cmd.log.Info("Check devspace.yaml for your configuration and make adjustments as needed")
	cmd.log.Infof("\r         \nYou can now run:\n- `%s` to pick which Kubernetes namespace to work in\n- `%s` to start developing your project in Kubernetes\n- `%s` to deploy your project to Kubernetes\n- `%s` to get a list of available commands", ansi.Color("devspace use namespace", "blue+b"), ansi.Color("devspace dev", "blue+b"), ansi.Color("devspace deploy", "blue+b"), ansi.Color("devspace -h", "blue+b"))
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

	if SpaceNameValidationRegEx.MatchString(dirname) == false || len(dirname) > 42 {
		dirname = "devspace"
	}

	return dirname, nil
}

func (cmd *InitCmd) addDevConfig(config *latest.Config) error {
	// Forward ports
	if len(config.Images) > 0 && len(config.Deployments) > 0 && config.Deployments[0].Helm != nil && config.Deployments[0].Helm.ComponentChart != nil && *config.Deployments[0].Helm.ComponentChart == true {
		componentValues := latest.ComponentConfig{}
		err := util.Convert(config.Deployments[0].Helm.Values, &componentValues)
		if err == nil && componentValues.Service != nil && componentValues.Service.Ports != nil && len(componentValues.Service.Ports) > 0 {
			servicePort := componentValues.Service.Ports[0]
			if servicePort.Port != nil {
				localPortPtr := servicePort.Port
				var remotePortPtr *int

				if *localPortPtr < 1024 {
					cmd.log.WriteString("\n")
					cmd.log.Warn("Your application listens on a system port [0-1024]. Choose a forwarding-port to access your application via localhost.")

					portString, err := cmd.log.Question(&survey.QuestionOptions{
						Question:     "Which forwarding port [1024-49151] do you want to use to access your application?",
						DefaultValue: strconv.Itoa(*localPortPtr + 8000),
					})
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
					{
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
					if len(ignoreRule) > 0 && ignoreRule[0] != "#"[0] && gitFolderIgnoreRegex.MatchString(ignoreRule) == false {
						excludePaths = append(excludePaths, ignoreRule)
					}
				}
			}

			syncConfig := &latest.SyncConfig{
				ImageName:          defaultImageName,
				UploadExcludePaths: excludePaths,
				ExcludePaths: []string{
					".git/",
				},
			}
			if config.Images[defaultImageName].InjectRestartHelper {
				syncConfig.OnUpload = &latest.SyncOnUpload{
					RestartContainer: true,
				}
			} else {
				config.Dev.Interactive = &latest.InteractiveConfig{
					DefaultEnabled: ptr.Bool(true),
				}
			}

			config.Dev.Sync = append(config.Dev.Sync, syncConfig)
		}
	}

	return nil
}

func (cmd *InitCmd) addProfileConfig(config *latest.Config) error {
	if len(config.Images) > 0 {
		defaultImageConfig, ok := (config.Images)[defaultImageName]
		if ok && (defaultImageConfig.Build == nil || defaultImageConfig.Build.Disabled == nil) {
			patchRemoveOp := "remove"
			patches := []*latest.PatchConfig{
				{
					Operation: patchRemoveOp,
					Path:      "images." + defaultImageName + ".appendDockerfileInstructions",
				},
			}

			if defaultImageConfig.InjectRestartHelper {
				patches = append(patches, &latest.PatchConfig{
					Operation: patchRemoveOp,
					Path:      "images." + defaultImageName + ".injectRestartHelper",
				})
			}
			if defaultImageConfig.RebuildStrategy != latest.RebuildStrategyDefault {
				patches = append(patches, &latest.PatchConfig{
					Operation: patchRemoveOp,
					Path:      "images." + defaultImageName + ".rebuildStrategy",
				})
			}
			if len(defaultImageConfig.Entrypoint) > 0 {
				patches = append(patches, &latest.PatchConfig{
					Operation: patchRemoveOp,
					Path:      "images." + defaultImageName + ".entrypoint",
				})
			}
			if defaultImageConfig.Build != nil && defaultImageConfig.Build.Docker != nil && defaultImageConfig.Build.Docker.Options != nil && defaultImageConfig.Build.Docker.Options.Target != "" {
				patches = append(patches, &latest.PatchConfig{
					Operation: patchRemoveOp,
					Path:      "images." + defaultImageName + ".build.docker.options.target",
				})
			}

			config.Profiles = append(config.Profiles, &latest.ProfileConfig{
				Name:    productionProfileName,
				Patches: patches,
			})
		}
		if ok && defaultImageConfig.InjectRestartHelper {
			config.Profiles = append(config.Profiles, &latest.ProfileConfig{
				Name: interactiveProfileName,
				Patches: []*latest.PatchConfig{
					{
						Operation: "add",
						Path:      "dev.interactive",
						Value: map[string]bool{
							"defaultEnabled": true,
						},
					},
					{
						Operation: "add",
						Path:      "images." + defaultImageName + ".entrypoint",
						Value: []string{
							"sleep",
							"9999999999",
						},
					},
				},
			})
		}
	}
	return nil
}
