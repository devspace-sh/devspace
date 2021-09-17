package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/plugin"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	latest "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/generator"
	"github.com/loft-sh/devspace/pkg/util/dockerfile"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SpaceNameValidationRegEx is the sapace name validation regex
var SpaceNameValidationRegEx = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$")

var gitFolderIgnoreRegex = regexp.MustCompile(`/?\.git/?`)

const gitIgnoreFile = ".gitignore"

const devspaceFolderGitignore = "\n\n# Ignore DevSpace cache and log folder\n.devspace/\n"

const (
	// Dockerfile not found options
	UseExistingDockerfileOption = "Use the Dockerfile in ./Dockerfile"
	CreateDockerfileOption      = "Create a Dockerfile for this project"
	EnterDockerfileOption       = "Enter path to a different Dockerfile"
	ComponentChartOption        = "helm: Use Component Helm Chart [QUICKSTART] (https://devspace.sh/component-chart/docs)"
	HelmChartOption             = "helm: Use my own Helm chart (e.g. local via ./chart/ or any remote chart)"
	ManifestsOption             = "kubectl: Use existing Kubernetes manifests (e.g. ./kube/deployment.yaml)"
	KustomizeOption             = "kustomize: Use an existing Kustomization (e.g. ./kube/kustomization/)"

	// The default name for the production profile
	productionProfileName = "production"
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
folder. Creates a devspace.yaml with all configuration.
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
	configLoader := f.NewConfigLoader("")
	configExists := configLoader.Exists()
	if configExists && !cmd.Reconfigure {
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
	err := plugin.ExecutePluginHook("init")
	if err != nil {
		return err
	}

	// Create config
	config := latest.New().(*latest.Config)
	generated, err := configLoader.LoadGenerated(nil)
	if err != nil {
		return err
	}

	// Create ConfigureManager
	configureManager := f.NewConfigureManager(config, generated, cmd.log)

	// Print DevSpace logo
	log.PrintLogo()

	// Add deployment and image config
	deploymentName, err := getDeploymentName()
	if err != nil {
		return err
	}

	imageName := "app"
	imageQuestion := ""
	selectedDeploymentOption := ""

	for {
		selectedDeploymentOption, err = cmd.log.Question(&survey.QuestionOptions{
			Question:     "How do you want to deploy this project?",
			DefaultValue: ComponentChartOption,
			Options: []string{
				ComponentChartOption,
				HelmChartOption,
				ManifestsOption,
				KustomizeOption,
			},
		})
		if err != nil {
			return err
		}

		if selectedDeploymentOption == HelmChartOption {
			imageQuestion = "What is the main container image of this project which is deployed by this Helm chart? (e.g. ecr.io/project/image)"
			err = configureManager.AddHelmDeployment(deploymentName)
		} else if selectedDeploymentOption == ManifestsOption || selectedDeploymentOption == KustomizeOption {
			if selectedDeploymentOption == ManifestsOption {
				imageQuestion = "What is the main container image of this project which is deployed by these manifests? (e.g. ecr.io/project/image)"
			} else {
				imageQuestion = "What is the main container image of this project which is deployed by this Kustomization? (e.g. ecr.io/project/image)"
			}
			err = configureManager.AddKubectlDeployment(deploymentName, selectedDeploymentOption == KustomizeOption)
		}

		if err != nil {
			if err.Error() != "" {
				cmd.log.WriteString("\n")
				cmd.log.Errorf("Error: %s", err.Error())
			}
		} else {
			break
		}
	}

	// Create new dockerfile generator
	dockerfileGenerator, err := generator.NewDockerfileGenerator("", "", cmd.log)
	if err != nil {
		return err
	}

	imageVarName := "IMAGE"
	imageVar := "${" + imageVarName + "}"

	for {
		image := ""
		if imageQuestion != "" {
			image, err = cmd.log.Question(&survey.QuestionOptions{
				Question:          imageQuestion,
				ValidationMessage: "Please enter a valid container image from a Kubernetes pod (e.g. myregistry.tld/project/image)",
				ValidationFunc: func(name string) error {
					_, _, err := imageselector.GetStrippedDockerImageName(strings.ToLower(name))
					return err
				},
			})
			if err != nil {
				return err
			}
		}

		err = configureManager.AddImage(imageName, image, cmd.Dockerfile, cmd.Context, dockerfileGenerator)
		if err != nil {
			if err.Error() != "" {
				cmd.log.Errorf("Error: %s", err.Error())
			}
		} else {
			break
		}
	}

	if config.Images != nil && config.Images[imageName] != nil {
		// Move full image name to variables
		config.Vars = append(config.Vars, &latest.Variable{
			Name:  imageVarName,
			Value: config.Images[imageName].Image,
		})

		// Use variable in images section
		config.Images[imageName].Image = imageVar
	}

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
	if selectedDeploymentOption == ComponentChartOption {
		err = configureManager.AddComponentDeployment(deploymentName, imageVar, port)
		if err != nil {
			return err
		}
	}

	// Add the development configuration
	err = cmd.addDevConfig(config, imageName, imageVar, port, dockerfileGenerator)
	if err != nil {
		return err
	}

	// Add the profile configuration
	err = cmd.addProfileConfig(config, imageName)
	if err != nil {
		return err
	}

	// Add .devspace/ to .gitignore
	err = appendToIgnoreFile(gitIgnoreFile, devspaceFolderGitignore)
	if err != nil {
		cmd.log.Warn(err)
	}

	// Save config
	err = configLoader.Save(config)
	if err != nil {
		return err
	}

	configPath := loader.ConfigPath("")
	annotatedConfig, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	configAnnotations := map[string]string{
		"(?m)^(vars:)":                    "\n# `vars` specifies variables which may be used as $${VAR_NAME} in devspace.yaml\n$1",
		"(?m)^(images:)":                  "\n# `images` specifies all images that may need to be built for this project\n$1",
		"(?m)^(  app:)":                   "$1 # This image is called `app` (you can have more than one image)",
		"(?m)^(deployments:)":             "\n# `deployments` tells DevSpace how to deploy this project\n$1",
		"(?m)^(  helm:)":                  "  # This deployment uses `helm` but you can also define `kubectl` deployments or kustomizations\n$1",
		"(?m)^(    )(componentChart:)":    "$1# We are deploying the so-called Component Chart: https://devspace.sh/component-chart/docs\n$1$2",
		"(?m)^(    )(chart:)":             "$1# We are deploying this project with the Helm chart you provided\n$1$2",
		"(?m)^(    )(values:)":            "$1# Under `values` we can define the values for this Helm chart used during `helm install/upgrade`\n$1# You may also use `valuesFiles` to load values from files, e.g. valuesFiles: [\"values.yaml\"]\n$1$2",
		"(?m)^(    )  someChartValue:.*":  "$1# image: $${IMAGE}\n$1# ingress:\n$1#   enabled: true",
		"(image: \\$\\{IMAGE\\})":         "$1 # Use the value of our `$${IMAGE}` variable here (see vars above)",
		"(?m)^(  kubectl:)":               "  # This deployment uses `kubectl` but you can also define `helm` deployments\n$1",
		"(?m)^(dev:)":                     "\n# `dev` only applies when you run `devspace dev`\n$1",
		"(?m)^(  ports:)":                 "  # `dev.ports` specifies all ports that should be forwarded while `devspace dev` is running\n  # Port-forwarding lets you access your application via localhost on your local machine\n$1",
		"(?m)^(  open:)":                  "\n  # `dev.open` tells DevSpace to open certain URLs as soon as they return HTTP status 200\n  # Since we configured port-forwarding, we can use a localhost address here to access our application\n$1",
		"(?m)^(  - url:.+)":               "$1\n",
		"(?m)^(  sync:)":                  "  # `dev.sync` configures a file sync between our Pods in k8s and your local project files\n$1",
		"(?m)^(  terminal:)":              "\n  # `dev.terminal` tells DevSpace to open a terminal as a last step during `devspace dev`\n$1",
		"(?m)^(    command:)":             "    # With this optional `command` we can tell DevSpace to run a script when opening the terminal\n    # This is often useful to display help info for new users or perform initial tasks (e.g. installing dependencies)\n    # DevSpace has generated an example ./devspace_start.sh file in your local project - Feel free to customize it!\n$1",
		"(?m)^(  replacePods:)":           "\n  # Since our Helm charts and manifests deployments are often optimized for production,\n  # DevSpace let's you swap out Pods dynamically to get a better dev environment\n$1",
		"(?m)^(    replaceImage:)":        "    # Since the `$${IMAGE}` used to start our main application pod may be distroless or not have any dev tooling, let's replace it with a dev-optimized image\n    # DevSpace provides a sample image here but you can use any image for your specific needs\n$1",
		"(?m)^(    patches:)":             "    # Besides replacing the container image, let's also apply some patches to the `spec` of our Pod\n    # We are overwriting `command` + `args` for the first container in our selected Pod, so it starts with `sleep 9999999`\n    # Using `sleep 9999999` as PID 1 (instead of the regular ENTRYPOINT), allows you to start the application manually\n$1",
		"(?m)^(  (-| ) imageSelector:.+)": "$1 # Select the Pod that runs our `$${IMAGE}`",
		"(?m)^(profiles:)":                "\n# `profiles` lets you modify the config above for different environments (e.g. dev vs production)\n$1",
		"(?m)^(- name: production)":       "  # This profile is called `production` and you can use it for example using: devspace deploy -p production\n  # We generally recommend to use the base config without any profiles as optimized for development (e.g. image build+push is disabled)\n$1",
		"(?m)^(  patches:)":               "# This profile applies patches to the config above.\n  # In this case, it enables image building for example by removing the `disabled: true` statement for the image `app`\n$1",
		"(?m)^(  merge:)":                 "# This profile adds our image to the config so that DevSpace will build, tag and push our image before the deployment\n$1",
	}

	for expr, replacement := range configAnnotations {
		annotatedConfig = regexp.MustCompile(expr).ReplaceAll(annotatedConfig, []byte(replacement))
	}

	err = ioutil.WriteFile(configPath, annotatedConfig, os.ModePerm)
	if err != nil {
		return err
	}

	// Save generated
	err = configLoader.SaveGenerated(generated)
	if err != nil {
		return errors.Errorf("Error saving generated file: %v", err)
	}

	cmd.log.WriteString("\n")
	cmd.log.Info("Configuration saved in devspace.yaml - you can make adjustments as needed")
	cmd.log.Done("Project successfully initialized")
	cmd.log.Infof("\r         \nYou can now run:\n- `%s` to pick which Kubernetes namespace to work in\n- `%s` to start developing your project in Kubernetes\n- `%s` to deploy your project to Kubernetes\n- `%s` to get a list of available commands", ansi.Color("devspace use namespace", "blue+b"), ansi.Color("devspace dev", "blue+b"), ansi.Color("devspace deploy -p production", "blue+b"), ansi.Color("devspace -h", "blue+b"))
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

	if !SpaceNameValidationRegEx.MatchString(dirname) || len(dirname) > 42 {
		dirname = "devspace"
	}

	return dirname, nil
}

func (cmd *InitCmd) addDevConfig(config *latest.Config, imageName, image string, port int, dockerfileGenerator *generator.DockerfileGenerator) error {
	// Forward ports
	if len(config.Deployments) > 0 {
		if port > 0 {
			localPort := port
			if localPort < 1024 {
				cmd.log.WriteString("\n")
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

			portMapping := latest.PortMapping{
				LocalPort: &port,
			}

			if port != localPort {
				portMapping = latest.PortMapping{
					LocalPort:  &localPort,
					RemotePort: &port,
				}
			}

			portMappings := []*latest.PortMapping{}
			portMappings = append(portMappings, &portMapping)

			// Add dev.ports config
			config.Dev.Ports = []*latest.PortForwardingConfig{
				{
					ImageSelector: image,
					PortMappings:  portMappings,
				},
			}

			// Add dev.open config
			config.Dev.Open = []*latest.OpenConfig{
				{
					URL: "http://localhost:" + strconv.Itoa(localPort),
				},
			}
		}

		// Specify sync path
		if config.Dev.Sync == nil {
			config.Dev.Sync = []*latest.SyncConfig{}
		}

		dockerignore, err := ioutil.ReadFile(".dockerignore")
		excludePaths := []string{}
		if err == nil {
			dockerignoreRules := strings.Split(string(dockerignore), "\n")
			for _, ignoreRule := range dockerignoreRules {
				ignoreRule = strings.TrimSpace(ignoreRule)
				if len(ignoreRule) > 0 && ignoreRule[0] != "#"[0] && !gitFolderIgnoreRegex.MatchString(ignoreRule) {
					excludePaths = append(excludePaths, ignoreRule)
				}
			}
		}

		syncConfig := &latest.SyncConfig{
			ImageSelector:      image,
			UploadExcludePaths: excludePaths,
			ExcludePaths: []string{
				".git/",
			},
		}

		if config.Images[imageName].InjectRestartHelper {
			syncConfig.OnUpload = &latest.SyncOnUpload{
				RestartContainer: true,
			}
		} else {
			fallbackLanguage := "alpine"
			language, err := dockerfileGenerator.GetLanguage()
			if err != nil {
				return err
			}

			if language == "none" {
				language = fallbackLanguage
			}

			if language == "java" {
				stat, err := os.Stat("build.gradle")
				if err == nil && !stat.IsDir() {
					language += "-gradle"
				} else {
					language += "-maven"
				}
			}

			startScriptName := "devspace_start.sh"
			startScriptContent, err := getScriptContent(language, startScriptName)
			if err != nil {
				// try fall back language
				startScriptContent, err = getScriptContent(fallbackLanguage, startScriptName)
				if err != nil {
					startScriptContent = []byte("#!/bin/bash\nbash")
				}

				language = fallbackLanguage
			}

			err = ioutil.WriteFile(startScriptName, startScriptContent, 0755)
			if err != nil {
				return err
			}

			config.Dev.Terminal = &latest.Terminal{
				ImageSelector: image,
				Command:       []string{"./" + startScriptName},
			}

			replacePodPatches := []*latest.PatchConfig{
				{
					Path:      "spec.containers[0].securityContext",
					Operation: "remove",
				},
			}

			if language != "php" {
				replacePodPatches = append([]*latest.PatchConfig{
					{
						Path:      "spec.containers[0].command",
						Operation: "replace",
						Value:     []string{"sleep"},
					},
					{
						Path:      "spec.containers[0].args",
						Operation: "replace",
						Value:     []string{"9999999"},
					},
				}, replacePodPatches...)
			}

			config.Dev.ReplacePods = []*latest.ReplacePod{
				{
					ImageSelector: image,
					ReplaceImage:  fmt.Sprintf("loftsh/%s:latest", language),
					Patches:       replacePodPatches,
				},
			}
		}

		config.Dev.Sync = append(config.Dev.Sync, syncConfig)
	}

	return nil
}

func getScriptContent(language, startScriptName string) ([]byte, error) {
	startFileURL := fmt.Sprintf("https://raw.githubusercontent.com/loft-sh/devtools-containers/main/%s/%s", language, startScriptName)
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err := client.Get(startFileURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return out, nil
	}

	return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(out))
}

func (cmd *InitCmd) addProfileConfig(config *latest.Config, imageName string) error {
	if len(config.Images) > 0 {
		imageConfig, ok := (config.Images)[imageName]
		if ok {
			profile := &latest.ProfileConfig{
				Name: productionProfileName,
			}

			// If image building is disabled, move it to production profile instead of disabling it
			if imageConfig.Build != nil && imageConfig.Build.Disabled {
				imageConfig.AppendDockerfileInstructions = []string{}
				imageConfig.InjectRestartHelper = false
				imageConfig.RebuildStrategy = latest.RebuildStrategyDefault
				imageConfig.Entrypoint = []string{}

				if imageConfig.Build.Docker != nil && imageConfig.Build.Docker.Options != nil && imageConfig.Build.Docker.Options.Target != "" {
					imageConfig.Build.Docker.Options.Target = ""
				}

				imageConfig.Build.Disabled = false
				if imageConfig.Build.Docker == nil && imageConfig.Build.BuildKit == nil && imageConfig.Build.Kaniko == nil {
					imageConfig.Build = nil
				}

				profile.Merge = &latest.ProfileConfigStructure{
					Images: map[interface{}]interface{}{
						imageName: imageConfig,
					},
				}

				delete(config.Images, imageName)
			} else {
				patchRemoveOp := "remove"
				patches := []*latest.PatchConfig{}

				if len(imageConfig.AppendDockerfileInstructions) > 0 {
					patches = append(patches, &latest.PatchConfig{
						Operation: patchRemoveOp,
						Path:      "images." + imageName + ".appendDockerfileInstructions",
					})
				}

				if imageConfig.InjectRestartHelper {
					patches = append(patches, &latest.PatchConfig{
						Operation: patchRemoveOp,
						Path:      "images." + imageName + ".injectRestartHelper",
					})
				}

				if imageConfig.RebuildStrategy != latest.RebuildStrategyDefault {
					patches = append(patches, &latest.PatchConfig{
						Operation: patchRemoveOp,
						Path:      "images." + imageName + ".rebuildStrategy",
					})
				}

				if len(imageConfig.Entrypoint) > 0 {
					patches = append(patches, &latest.PatchConfig{
						Operation: patchRemoveOp,
						Path:      "images." + imageName + ".entrypoint",
					})
				}

				if imageConfig.Build != nil && imageConfig.Build.Docker != nil && imageConfig.Build.Docker.Options != nil && imageConfig.Build.Docker.Options.Target != "" {
					patches = append(patches, &latest.PatchConfig{
						Operation: patchRemoveOp,
						Path:      "images." + imageName + ".build.docker.options.target",
					})
				}

				if len(patches) == 0 {
					return nil
				}

				profile.Patches = patches
			}

			config.Profiles = append(config.Profiles, profile)

		}
	}
	return nil
}
