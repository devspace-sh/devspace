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

	// Create config
	localCache, err := localcache.NewCacheLoader().Load(constants.DefaultConfigPath)
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
	config := latest.New().(*latest.Config)
	if err != nil {
		return err
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

	// Create new dockerfile generator
	languageHandler, err := generator.NewLanguageHandler("", "", cmd.log)
	if err != nil {
		return err
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

	config.Commands = map[string]*latest.CommandConfig{
		"migrate-db": {
			Command: `echo 'This is a cross-platform, shared command that can be used to codify any kind of dev task.'
echo 'Anyone using this project can invoke it via "devspace run migrate-db"'`,
		},
	}

	// Add pipeline: dev
	config.Pipelines = map[string]*latest.Pipeline{
		"dev": {
			Run: `run_dependency_pipelines --all    # 1. Deploy any projects this project needs (see "dependencies")
create_deployments --all          # 2. Deploy Helm charts and manifests specfied as "deployments"
start_dev ` + imageName + `                     # 3. Start dev mode "` + imageName + `" (see "dev" section)`,
		},
		"deploy": {
			Run: `run_dependency_pipelines --all                    # 1. Deploy any projects this project needs (see "dependencies")
build_images --all -t $(git describe --always)    # 2. Build, tag (git commit hash) and push all images (see "images")
create_deployments --all \                        # 3. Deploy Helm charts and manifests specfied as "deployments"
  --set updateImageTags=true                      #    + make sure to update all image tags to the one from step 2`,
		},
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

	configAnnotations := map[string]string{
		"(?m)^(vars:)":                               "\n# `vars` specifies variables which may be used as $${VAR_NAME} in devspace.yaml\n$1",
		"(?m)^(images:)":                             "\n# `images` specifies all images that may need to be built for this project\n$1",
		"(?m)^(deployments:)":                        "\n# `deployments` tells DevSpace how to deploy this project\n$1",
		"(?m)^(  helm:)":                             "  # This deployment uses `helm` but you can also define `kubectl` deployments or kustomizations\n$1",
		"(?m)^(    )(componentChart:)":               "$1# We are deploying the so-called Component Chart: https://devspace.sh/component-chart/docs\n$1$2",
		"(?m)^(    )(chart:)":                        "$1# We are deploying this project with the Helm chart you provided\n$1$2",
		"(?m)^(    )(values:)":                       "$1# Under `values` we can define the values for this Helm chart used during `helm install/upgrade`\n$1# You may also use `valuesFiles` to load values from files, e.g. valuesFiles: [\"values.yaml\"]\n$1$2",
		"(?m)^(    )  someChartValue:.*":             "$1# image: $${IMAGE}\n$1# ingress:\n$1#   enabled: true",
		"(image: \\$\\{IMAGE\\})":                    "$1 # Use the value of our `$${IMAGE}` variable here (see vars above)",
		"(?m)^(  kubectl:)":                          "  # This deployment uses `kubectl` but you can also define `helm` deployments\n$1",
		"(?m)^(dev:)":                                "\n# `dev` only applies when you run `devspace dev`\n$1",
		"(?m)^(  ports:)":                            "  # `dev.ports` specifies all ports that should be forwarded while `devspace dev` is running\n  # Port-forwarding lets you access your application via localhost on your local machine\n$1",
		"(?m)^(  open:)":                             "\n  # `dev.open` tells DevSpace to open certain URLs as soon as they return HTTP status 200\n  # Since we configured port-forwarding, we can use a localhost address here to access our application\n$1",
		"(?m)^(  - url:.+)":                          "$1\n",
		"(?m)^(  sync:)":                             "  # `dev.sync` configures a file sync between our Pods in k8s and your local project files\n$1",
		"(?m)^(  (-| ) excludePaths:)":               "    # `excludePaths` option expects an array of strings with paths that should not be synchronized between the\n    # local filesystem and the remote container filesystem. It uses the same syntax as `.gitignore`.\n$1",
		"(?m)^(  terminal:)":                         "\n  # `dev.terminal` tells DevSpace to open a terminal as a last step during `devspace dev`\n$1",
		"(?m)^(    command:)":                        "    # With this optional `command` we can tell DevSpace to run a script when opening the terminal\n    # This is often useful to display help info for new users or perform initial tasks (e.g. installing dependencies)\n    # DevSpace has generated an example ./devspace_start.sh file in your local project - Feel free to customize it!\n$1",
		"(?m)^(  replacePods:)":                      "\n  # Since our Helm charts and manifests deployments are often optimized for production,\n  # DevSpace let's you swap out Pods dynamically to get a better dev environment\n$1",
		"(?m)^(    replaceImage:)":                   "    # Since the `$${IMAGE}` used to start our main application pod may be distroless or not have any dev tooling, let's replace it with a dev-optimized image\n    # DevSpace provides a sample image here but you can use any image for your specific needs\n$1",
		"(?m)^(    patches:)":                        "    # Besides replacing the container image, let's also apply some patches to the `spec` of our Pod\n    # We are overwriting `command` + `args` for the first container in our selected Pod, so it starts with `sleep 9999999`\n    # Using `sleep 9999999` as PID 1 (instead of the regular ENTRYPOINT), allows you to start the application manually\n$1",
		"(?m)^(  (-| ) imageSelector:\\s?([^\\s]+))": "$1 # Select the Pod that runs our `$3`",
		"(?m)^(profiles:)":                           "\n# `profiles` lets you modify the config above for different environments (e.g. dev vs production)\n$1",
		"(?m)^(- name: production)":                  "  # This profile is called `production` and you can use it for example using: devspace deploy -p production\n  # We generally recommend using the base config without any profiles as optimized for development (e.g. image build+push is disabled)\n$1",
		"(?m)^(  patches:)":                          "# This profile applies patches to the config above.\n  # In this case, it enables image building for example by removing the `disabled: true` statement for the image `app`\n$1",
		"(?m)^(  merge:)":                            "# This profile adds our image to the config so that DevSpace will build, tag and push our image before the deployment\n$1",
	}

	for expr, replacement := range configAnnotations {
		annotatedConfig = regexp.MustCompile(expr).ReplaceAll(annotatedConfig, []byte(replacement))
	}

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
		projectParts := strings.Split(string(regexp.MustCompile("^.*://github.com/(.*?)(.?git)?").ReplaceAll(gitRemote, []byte("$1"))), sep)
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

	devConfig := &latest.DevPod{
		ImageSelector: image,
	}

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
				Port: fmt.Sprintf("%d:%d", &localPort, &port),
			}
		}
		devConfig.Ports = []*latest.PortMapping{&portMapping}

		// Add dev.open config
		devConfig.Open = []*latest.OpenConfig{
			{
				URL: "http://localhost:" + strconv.Itoa(localPort),
			},
		}
	}

	devConfig.Sync = []*latest.SyncConfig{
		{
			Path: "./",
		},
	}

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

	err = languageHandler.CopyFile(startScriptName, startScriptName, false)
	if err != nil {
		return err
	}

	devImage, err := languageHandler.GetDevImage()
	if err != nil {
		return err
	}

	devConfig.DevImage = devImage

	// Add dev section to config
	config.Dev[imageName] = devConfig

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
