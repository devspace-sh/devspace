package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	yaml "gopkg.in/yaml.v3"

	"github.com/loft-sh/devspace/pkg/devspace/hook"

	"github.com/loft-sh/devspace/pkg/devspace/plugin"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	latest "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/generator"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/dockerfile"
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

const gitIgnoreFile = ".gitignore"
const startScriptName = "devspace_start.sh"

const devspaceFolderGitignore = "\n\n# Ignore DevSpace cache and log folder\n.devspace/\n"

const (
	// Dockerfile not found options
	UseExistingDockerfileOption       = "Use the Dockerfile in ./Dockerfile"
	CreateDockerfileOption            = "Create a Dockerfile for this project"
	EnterDockerfileOption             = "Enter path to a different Dockerfile"
	DeployOptionHelm                  = "helm"
	DeployOptionKubectl               = "kubectl"
	DeployOptionKustomize             = "kustomize"
	NewDevSpaceConfigOption           = "Create a new devspace.yaml from scratch"
	DockerComposeDevSpaceConfigOption = "Convert existing docker-compose.yml to devspace.yaml"
)

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	// Flags
	Reconfigure bool
	Dockerfile  string
	Context     string
	Provider    string
	log         log.Logger
}

// NewInitCmd creates a new init command
func NewInitCmd(f factory.Factory) *cobra.Command {
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
folder. Creates a devspace.yaml as a starting point.
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	initCmd.Flags().BoolVarP(&cmd.Reconfigure, "reconfigure", "r", false, "Change existing configuration")
	initCmd.Flags().StringVar(&cmd.Context, "context", "", "Context path to use for intialization")
	initCmd.Flags().StringVar(&cmd.Dockerfile, "dockerfile", helper.DefaultDockerfilePath, "Dockerfile to use for initialization")
	initCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	return initCmd
}

// Run executes the command logic
func (cmd *InitCmd) Run(f factory.Factory) error {
	// Check if config already exists
	cmd.log = f.GetLog()
	configLoader, err := f.NewConfigLoader("")
	if err != nil {
		return err
	}
	configExists := configLoader.Exists()
	if configExists && !cmd.Reconfigure {
		optionNo := "No"
		cmd.log.WriteString(cmd.log.GetLevel(), "\n")
		cmd.log.Warnf("%s already exists in this project", ansi.Color("devspace.yaml", "white+b"))
		response, err := cmd.log.Question(&survey.QuestionOptions{
			Question: "Do you want to delete devspace.yaml and recreate it from scratch?",
			Options:  []string{optionNo, "Yes"},
		})
		if err != nil {
			return err
		}

		if response == optionNo {
			return nil
		}
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
	err = hook.ExecuteHooks(nil, nil, "init")
	if err != nil {
		return err
	}

	// Print DevSpace logo
	log.PrintLogo()

	/*
		    generateFromDockerCompose := false
			// TODO: Enable again
			dockerComposePath := "" // compose.GetDockerComposePath()
			if dockerComposePath != "" {
				selectedDockerComposeOption, err := cmd.log.Question(&survey.QuestionOptions{
					Question:     "Docker Compose configuration detected. Do you want to create a DevSpace configuration based on Docker Compose?",
					DefaultValue: DockerComposeDevSpaceConfigOption,
					Options: []string{
						DockerComposeDevSpaceConfigOption,
						NewDevSpaceConfigOption,
					},
				})
				if err != nil {
					return err
				}

				generateFromDockerCompose = selectedDockerComposeOption == DockerComposeDevSpaceConfigOption
			}

			if generateFromDockerCompose {
				composeLoader := compose.NewDockerComposeLoader(dockerComposePath)
				if err != nil {
					return err
				}

				// Load config
				config, err := composeLoader.Load(cmd.log)
				if err != nil {
					return err
				}

				// Save config
				err = composeLoader.Save(config)
				if err != nil {
					return err
				}
			} else {*/

	// Create new dockerfile generator
	languageHandler, err := generator.NewLanguageHandler("", "", cmd.log)
	if err != nil {
		return err
	}

	err = languageHandler.CopyTemplates(".", false)
	if err != nil {
		return err
	}

	startScriptAbsPath, err := filepath.Abs(startScriptName)
	if err != nil {
		return err
	}

	_, err = os.Stat(startScriptAbsPath)
	if err == nil {
		// Ensure file is executable
		err = os.Chmod(startScriptAbsPath, 0755)
		if err != nil {
			return err
		}
	}

	var config *latest.Config

	// create kubectl client
	client, err := f.NewKubeClientFromContext(globalFlags.KubeContext, globalFlags.Namespace)
	if err == nil {
		configInterface, err := configLoader.Load(context.TODO(), client, &loader.ConfigOptions{}, cmd.log)
		if err == nil {
			config = configInterface.Config()
		}
	}

	localCache, err := localcache.NewCacheLoader().Load(constants.DefaultConfigPath)
	if err != nil {
		return err
	}

	if config == nil {
		// Create config
		config = latest.New().(*latest.Config)
		if err != nil {
			return err
		}
	}

	// Create ConfigureManager
	configureManager := f.NewConfigureManager(config, localCache, cmd.log)

	// Determine name for this devspace project
	projectName, projectNamespace, err := getProjectName()
	if err != nil {
		return err
	}

	config.Name = projectName

	imageName := "app"
	selectedDeploymentOption := ""
	mustAddComponentChart := false

	for {
		selectedDeploymentOption, err = cmd.log.Question(&survey.QuestionOptions{
			Question: "How do you want to deploy this project?",
			Options: []string{
				DeployOptionHelm,
				DeployOptionKubectl,
				DeployOptionKustomize,
			},
		})
		if err != nil {
			return err
		}

		isQuickstart := strings.HasPrefix(projectName, "devspace-quickstart-")

		if selectedDeploymentOption != DeployOptionHelm && isQuickstart {
			cmd.log.WriteString(logrus.InfoLevel, "\n")
			cmd.log.Warn("If this is a DevSpace quickstart project, you should use Helm!")

			useHelm := "Yes"
			helmAnswer, err := cmd.log.Question(&survey.QuestionOptions{
				Question: "Do you want to switch to using Helm as suggested?",
				Options: []string{
					useHelm,
					"No",
				},
			})
			if err != nil {
				return err
			}

			if helmAnswer == useHelm {
				selectedDeploymentOption = DeployOptionHelm
			}
		}

		if selectedDeploymentOption == DeployOptionHelm {
			hasOwnHelmChart := "Yes"
			helmChartAnswer, err := cmd.log.Question(&survey.QuestionOptions{
				Question: "Do you already have a Helm chart for this project?",
				Options: []string{
					"No",
					hasOwnHelmChart,
				},
			})
			if err != nil {
				return err
			}

			if isQuickstart {
				quickstartYes := "Yes"
				quickstartAnswer, err := cmd.log.Question(&survey.QuestionOptions{
					Question: "Is this a DevSpace Quickstart project?",
					Options: []string{
						quickstartYes,
						"No",
					},
				})
				if err != nil {
					return err
				}

				if quickstartAnswer == quickstartYes {
					mustAddComponentChart = true
				}
			}

			if helmChartAnswer == hasOwnHelmChart && !mustAddComponentChart {
				err = configureManager.AddHelmDeployment(imageName)
				if err != nil {
					if err.Error() != "" {
						cmd.log.WriteString(logrus.InfoLevel, "\n")
						cmd.log.Errorf("Error: %s", err.Error())
					}

					// Retry questions on error
					continue
				}
			} else {
				mustAddComponentChart = true
			}
		} else if selectedDeploymentOption == DeployOptionKubectl || selectedDeploymentOption == DeployOptionKustomize {
			err = configureManager.AddKubectlDeployment(imageName, selectedDeploymentOption == DeployOptionKustomize)
			if err != nil {
				if err.Error() != "" {
					cmd.log.WriteString(logrus.InfoLevel, "\n")
					cmd.log.Errorf("Error: %s", err.Error())
				}

				// Retry questions on error
				continue
			}
		}
		break
	}

	image := ""
	for {
		if !mustAddComponentChart {
			manifests, err := cmd.render(f, config)
			if err != nil {
				return errors.Wrap(err, "error rendering deployment")
			}

			images, err := parseImages(manifests)
			if err != nil {
				return errors.Wrap(err, "error parsing images")
			}

			if len(images) == 0 {
				return fmt.Errorf("no images found for the selected deployments")
			}

			image, err = cmd.log.Question(&survey.QuestionOptions{
				Question:     "Which image do you want to develop with DevSpace?",
				DefaultValue: images[0],
				Options:      images,
			})
			if err != nil {
				return err
			}
		}

		err = configureManager.AddImage(imageName, image, projectNamespace+"/"+projectName, cmd.Dockerfile, languageHandler)
		if err != nil {
			if err.Error() != "" {
				cmd.log.Errorf("Error: %s", err.Error())
			}
		} else {
			break
		}
	}

	image = config.Images[imageName].Image

	// Determine app port
	portString := ""

	// Try to get ports from dockerfile
	ports, err := dockerfile.GetPorts(config.Images[imageName].Dockerfile)
	if err == nil {
		if len(ports) == 1 {
			portString = strconv.Itoa(ports[0])
		} else if len(ports) > 1 {
			portString, err = cmd.log.Question(&survey.QuestionOptions{
				Question:     "Which port is your application listening on?",
				DefaultValue: strconv.Itoa(ports[0]),
			})
			if err != nil {
				return err
			}

			if portString == "" {
				portString = strconv.Itoa(ports[0])
			}
		}
	}

	if portString == "" {
		portString, err = cmd.log.Question(&survey.QuestionOptions{
			Question:               "Which port is your application listening on? (Enter to skip)",
			ValidationRegexPattern: "[0-9]*",
		})
		if err != nil {
			return err
		}
	}

	port := 0
	if portString != "" {
		port, err = strconv.Atoi(portString)
		if err != nil {
			return errors.Wrap(err, "error parsing port")
		}
	}

	// Add component deployment if selected
	if mustAddComponentChart {
		err = configureManager.AddComponentDeployment(imageName, image, port)
		if err != nil {
			return err
		}
	}

	// Add the development configuration
	err = cmd.addDevConfig(config, imageName, image, port, languageHandler)
	if err != nil {
		return err
	}

	if config.Commands == nil {
		config.Commands = map[string]*latest.CommandConfig{}

		config.Commands["migrate-db"] = &latest.CommandConfig{
			Command: `echo 'This is a cross-platform, shared command that can be used to codify any kind of dev task.'
	echo 'Anyone using this project can invoke it via "devspace run migrate-db"'`,
		}
	}

	if config.Pipelines == nil {
		config.Pipelines = map[string]*latest.Pipeline{}
	}

	// Add pipeline: dev
	config.Pipelines["dev"] = &latest.Pipeline{
		Run: `run_dependency_pipelines --all    # 1. Deploy any projects this project needs (see "dependencies")
create_deployments --all          # 2. Deploy Helm charts and manifests specfied as "deployments"
start_dev ` + imageName + `                     # 3. Start dev mode "` + imageName + `" (see "dev" section)`,
	}

	// Add pipeline: dev
	config.Pipelines["deploy"] = &latest.Pipeline{
		Run: `run_dependency_pipelines --all                    # 1. Deploy any projects this project needs (see "dependencies")
build_images --all -t $(git describe --always)    # 2. Build, tag (git commit hash) and push all images (see "images")
create_deployments --all \                        # 3. Deploy Helm charts and manifests specfied as "deployments"
  --set updateImageTags=true                      #    + make sure to update all image tags to the one from step 2`,
	}

	// Save config
	err = loader.Save(constants.DefaultConfigPath, config)
	if err != nil {
		return err
	}

	/*}*/

	// Save generated
	err = localCache.Save()
	if err != nil {
		return errors.Errorf("Error saving generated file: %v", err)
	}

	// Add .devspace/ to .gitignore
	err = appendToIgnoreFile(gitIgnoreFile, devspaceFolderGitignore)
	if err != nil {
		cmd.log.Warn(err)
	}

	configPath := loader.ConfigPath("")
	annotatedConfig, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	annotatedConfig = regexp.MustCompile("(?m)(\n\\s{2,6}name:.*)").ReplaceAll(annotatedConfig, []byte(""))
	annotatedConfig = regexp.MustCompile("(?s)(\n  deploy:.*)(\n  dev:.*)(\nimages:)").ReplaceAll(annotatedConfig, []byte("$2$1$3"))
	annotatedConfig = regexp.MustCompile("(?s)(\n    imageSelector:.*?)(\n.*)(\n    devImage:.*?)(\n)").ReplaceAll(annotatedConfig, []byte("$1$3$2$4"))

	configAnnotations := map[string]string{
		"(?m)^(pipelines:)":           "\n# This is a list of `pipelines` that DevSpace can execute (you can define your own)\n$1",
		"(?m)^(  )(deploy:)":          "$1# You can run this pipeline via `devspace deploy` (or `devspace run-pipeline deploy`)\n$1$2",
		"(?m)^(  )(dev:)":             "$1# This is the pipeline for the main command: `devspace dev` (or `devspace run-pipeline dev`)\n$1$2",
		"(?m)^(images:)":              "\n# This is a list of `images` that DevSpace can build for this project\n# We recommend to skip image building during development (devspace dev) as much as possible\n$1",
		"(?m)^(deployments:)":         "\n# This is a list of `deployments` that DevSpace can create for this project\n$1",
		"(?m)^(    )(helm:)":          "$1# This deployment uses `helm` but you can also define `kubectl` deployments or kustomizations\n$1$2",
		"(?m)^(      )(chart:)":       "$1# We are deploying this project with the Helm chart you provided\n$1$2",
		"(?m)^(      )(values:)":      "$1# Under `values` we can define the values for this Helm chart used during `helm install/upgrade`\n$1# You may also use `valuesFiles` to load values from files, e.g. valuesFiles: [\"values.yaml\"]\n$1$2",
		"(?m)^(    )(kubectl:)":       "$1# This deployment uses `kubectl` but you can also define `helm` deployments\n$1$2",
		"(?m)^(dev:)":                 "\n# This is a list of `dev` containers that are based on the containers created by your deployments\n$1",
		"(?m)^(    )(imageSelector:)": "$1# Search for the container that runs this image\n$1$2",
		"(?m)^(    )(devImage:)":      "$1# Replace the container image with this dev-optimized image (allows to skip image building during development)\n$1$2",
		"(?m)^(    )(sync:)":          "$1# Sync files between the local filesystem and the development container\n$1$2",
		"(?m)^(    )(ports:)":         "$1# Forward the following ports to be able access your application via localhost\n$1$2",
		"(?m)^(    )(open:)":          "$1# Open the following URLs once they return an HTTP status code other than 502 or 503\n$1$2",
		"(?m)^(    )(terminal:)":      "$1# Open a terminal and use the following command to start it\n$1$2",
		"(?m)^(    )(ssh:)":           "$1# Inject a lightweight SSH server into the container (so your IDE can connect to the remote dev env)\n$1$2",
		"(?m)^(    )(proxyCommands:)": "$1# Make the following commands from my local machine available inside the dev container\n$1$2",
		"(?m)^(commands:)":            "\n# Use the `commands` section to define repeatable dev workflows for this project \n$1",
	}

	for expr, replacement := range configAnnotations {
		annotatedConfig = regexp.MustCompile(expr).ReplaceAll(annotatedConfig, []byte(replacement))
	}

	annotatedConfig = append(annotatedConfig, []byte(`
# Define dependencies to other projects with a devspace.yaml
# dependencies:
#   api:
#     git: https://...  # Git-based dependencies
#     tag: v1.0.0
#   ui:
#     path: ./ui        # Path-based dependencies (for monorepos)
`)...)

	err = ioutil.WriteFile(configPath, annotatedConfig, os.ModePerm)
	if err != nil {
		return err
	}

	cmd.log.WriteString(logrus.InfoLevel, "\n")
	cmd.log.Done("Project successfully initialized")
	cmd.log.Info("Configuration saved in devspace.yaml - you can make adjustments as needed")
	cmd.log.Infof("\r         \nYou can now run:\n1. %s - to pick which Kubernetes namespace to work in\n2. %s - to start developing your project in Kubernetes\n\nRun `%s` or `%s` to see a list of available commands and flags\n", ansi.Color("devspace use namespace", "blue+b"), ansi.Color("devspace dev", "blue+b"), ansi.Color("devspace -h", "blue+b"), ansi.Color("devspace [command] -h", "blue+b"))
	return nil
}

func appendToIgnoreFile(ignoreFile, content string) error {
	// Check if ignoreFile exists
	_, err := os.Stat(ignoreFile)
	if os.IsNotExist(err) {
		_ = fsutil.WriteToFile([]byte(content), ignoreFile)
	} else {
		fileContent, err := ioutil.ReadFile(ignoreFile)
		if err != nil {
			return errors.Errorf("Error reading file %s: %v", ignoreFile, err)
		}

		// append only if not found in file content
		if !strings.Contains(string(fileContent), content) {
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

func getProjectName() (string, string, error) {
	projectName := ""
	projectNamespace := ""
	gitRemote, err := command.Output(context.TODO(), "", "git", "config", "--get", "remote.origin.url")
	if err == nil {
		sep := "/"
		projectParts := strings.Split(string(regexp.MustCompile("^.*?://[^/]+?/([^.]+)(\\.git)?").ReplaceAll(gitRemote, []byte("$1"))), sep)
		partsLen := len(projectParts)
		if partsLen > 1 {
			projectNamespace = strings.Join(projectParts[0:partsLen-1], sep)
			projectName = projectParts[partsLen-1]
		}
	}

	if projectName == "" {
		absPath, err := filepath.Abs(".")
		if err != nil {
			return "", "", err
		}
		projectName = filepath.Base(absPath)
	}

	projectName = strings.ToLower(projectName)
	projectName = regexp.MustCompile("[^a-zA-Z0-9- ]+").ReplaceAllString(projectName, "")
	projectName = regexp.MustCompile("[^a-zA-Z0-9-]+").ReplaceAllString(projectName, "-")
	projectName = strings.Trim(projectName, "-")

	if !SpaceNameValidationRegEx.MatchString(projectName) || len(projectName) > 42 {
		projectName = "devspace"
	}

	return projectName, projectNamespace, nil
}

func (cmd *InitCmd) addDevConfig(config *latest.Config, imageName, image string, port int, languageHandler *generator.LanguageHandler) error {
	if config.Dev == nil {
		config.Dev = map[string]*latest.DevPod{}
	}

	devConfig, ok := config.Dev[imageName]
	if !ok {
		devConfig = &latest.DevPod{}
		config.Dev[imageName] = devConfig
	}

	devConfig.ImageSelector = image

	if port > 0 {
		localPort := port
		if localPort < 1024 {
			cmd.log.WriteString(logrus.InfoLevel, "\n")
			cmd.log.Warn("Your application listens on a system port [0-1024]. Choose a forwarding-port to access your application via localhost.")

			portString, err := cmd.log.Question(&survey.QuestionOptions{
				Question:     "Which forwarding port [1024-49151] do you want to use to access your application?",
				DefaultValue: strconv.Itoa(localPort + 8000),
			})
			if err != nil {
				return err
			}

			localPort, err = strconv.Atoi(portString)
			if err != nil {
				return errors.Errorf("Error parsing port '%s'", portString)
			}
		}

		// Add dev.ports
		portMapping := latest.PortMapping{
			Port: fmt.Sprintf("%d", port),
		}
		if port != localPort {
			portMapping = latest.PortMapping{
				Port: fmt.Sprintf("%d:%d", localPort, port),
			}
		}

		if devConfig.Ports == nil {
			devConfig.Ports = []*latest.PortMapping{}
		}
		devConfig.Ports = append(devConfig.Ports, &portMapping)

		if devConfig.Open == nil {
			devConfig.Open = []*latest.OpenConfig{}
		}
		devConfig.Open = append(devConfig.Open, &latest.OpenConfig{
			URL: "http://localhost:" + strconv.Itoa(localPort),
		})
	}

	if devConfig.Sync == nil {
		devConfig.Sync = []*latest.SyncConfig{}
	}

	devConfig.Sync = append(devConfig.Sync, &latest.SyncConfig{
		Path: "./",
	})

	devConfig.Terminal = &latest.Terminal{
		Command: "./" + startScriptName,
	}

	// Determine language
	language, err := languageHandler.GetLanguage()
	if err != nil {
		return err
	}

	if language == "java" {
		stat, err := os.Stat("build.gradle")
		if err == nil && !stat.IsDir() {
			language += "-gradle"
		} else {
			language += "-maven"
		}
	}

	devImage, err := languageHandler.GetDevImage()
	if err != nil {
		return err
	}

	devConfig.DevImage = devImage

	devConfig.SSH = &latest.SSH{
		Enabled: true,
	}

	if devConfig.ProxyCommands == nil {
		devConfig.ProxyCommands = []*latest.ProxyCommand{}
	}

	devConfig.ProxyCommands = append(devConfig.ProxyCommands, []*latest.ProxyCommand{
		{
			Command: "devspace",
		},
		{
			Command: "kubectl",
		},
		{
			Command: "helm",
		},
		{
			Command: "git",
		},
	}...)

	return nil
}

func (cmd *InitCmd) render(f factory.Factory, config *latest.Config) (string, error) {
	// Save temporary file to render it
	renderPath := loader.ConfigPath("render.yaml")
	err := loader.Save(renderPath, config)
	defer os.Remove(renderPath)
	if err != nil {
		return "", err
	}

	// Use the render command to render it.
	writer := &bytes.Buffer{}
	renderCmd := &RenderCmd{
		GlobalFlags: &flags.GlobalFlags{
			Silent:     true,
			ConfigPath: renderPath,
		},
		SkipPush:  true,
		SkipBuild: true,
		Writer:    writer,
	}
	err = renderCmd.Run(f)
	if err != nil {
		return "", err
	}

	return writer.String(), nil
}

func parseImages(manifests string) ([]string, error) {
	images := []string{}

	var doc yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader([]byte(manifests)))
	for dec.Decode(&doc) == nil {
		path, err := yamlpath.NewPath("..image")
		if err != nil {
			return nil, err
		}

		matches, err := path.Find(&doc)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			images = append(images, match.Value)
		}
	}

	return images, nil
}
