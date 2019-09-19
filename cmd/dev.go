package cmd

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/dependency"
	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/devspace/watch"
	"github.com/mgutz/ansi"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/util/exit"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	logutil "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DevCmd is a struct that defines a command call for "up"
type DevCmd struct {
	*flags.GlobalFlags

	SkipPush                bool
	AllowCyclicDependencies bool
	VerboseDependencies     bool
	SkipOpen                bool

	ForceBuild        bool
	SkipBuild         bool
	BuildSequential   bool
	ForceDeploy       bool
	Deployments       string
	ForceDependencies bool

	Sync            bool
	Terminal        bool
	ExitAfterDeploy bool
	SkipPipeline    bool
	Portforwarding  bool
	VerboseSync     bool
	Interactive     bool
}

const interactiveDefaultPickerValue = "Open Picker"

// NewDevCmd creates a new devspace dev command
func NewDevCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DevCmd{GlobalFlags: globalFlags}

	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Starts the development mode",
		Long: `
#######################################################
################### devspace dev ######################
#######################################################
Starts your project in development mode:
1. Builds your Docker images and override entrypoints if specified
2. Deploys the deployments via helm or kubectl
3. Forwards container ports to the local computer
4. Starts the sync client
5. Streams the logs of deployed containers

Use Interactive Mode:
- Use "devspace dev -i" for interactive mode (terminal)
#######################################################`,
		RunE: cmd.Run,
	}

	devCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")
	devCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", false, "Deploys the dependencies verbosely")

	devCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	devCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	devCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")

	devCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to deploy every deployment")
	devCmd.Flags().StringVar(&cmd.Deployments, "deployments", "", "Only deploy a specifc deployment (You can specify multiple deployments comma-separated")
	devCmd.Flags().BoolVar(&cmd.ForceDependencies, "force-dependencies", false, "Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)")

	devCmd.Flags().BoolVarP(&cmd.SkipPipeline, "skip-pipeline", "x", false, "Skips build & deployment and only starts sync, portforwarding & terminal")
	devCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")

	devCmd.Flags().BoolVar(&cmd.Sync, "sync", true, "Enable code synchronization")
	devCmd.Flags().BoolVar(&cmd.VerboseSync, "verbose-sync", false, "When enabled the sync will log every file change")

	devCmd.Flags().BoolVar(&cmd.Portforwarding, "portforwarding", true, "Enable port forwarding")

	devCmd.Flags().BoolVar(&cmd.ExitAfterDeploy, "exit-after-deploy", false, "Exits the command after building the images and deploying the project")
	devCmd.Flags().BoolVarP(&cmd.Interactive, "interactive", "i", false, "Enable interactive mode for images (overrides entrypoint with sleep command) and start terminal proxy")
	return devCmd
}

// Run executes the command logic
func (cmd *DevCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Start file logging
	log.StartFileLogging()

	// Validate flags
	err = cmd.validateFlags()
	if err != nil {
		return err
	}

	// Load generated config
	generatedConfig, err := generated.LoadConfig(cmd.Profile)
	if err != nil {
		return errors.Errorf("Error loading generated.yaml: %v", err)
	}

	// Create kubectl client and switch context if specified
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, true, log.GetInstance())
	if err != nil {
		return err
	}

	// Get the config
	config, err := cmd.loadConfig()
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		return err
	}

	// Create namespace if necessary
	err = client.EnsureDefaultNamespace(log.GetInstance())
	if err != nil {
		return errors.Errorf("Unable to create namespace: %v", err)
	}

	// Create cluster role binding if necessary
	err = client.EnsureGoogleCloudClusterRoleBinding(log.GetInstance())
	if err != nil {
		return err
	}

	// Create the image pull secrets and add them to the default service account
	dockerClient, err := docker.NewClient(log.GetInstance())
	if err != nil {
		dockerClient = nil
	}

	err = registry.CreatePullSecrets(config, client, dockerClient, log.GetInstance())
	if err != nil {
		return err
	}

	// Build and deploy images
	exitCode, err := cmd.buildAndDeploy(config, generatedConfig, client, args, true)
	if err != nil {
		return err
	} else if exitCode != 0 {
		exit.Exit(exitCode)
	}

	return nil
}

func (cmd *DevCmd) buildAndDeploy(config *latest.Config, generatedConfig *generated.Config, client *kubectl.Client, args []string, skipBuildIfAlreadyBuilt bool) (int, error) {
	if cmd.SkipPipeline == false {
		// Dependencies
		err := dependency.DeployAll(config, generatedConfig, client, cmd.AllowCyclicDependencies, false, cmd.SkipPush, cmd.ForceDependencies, cmd.SkipBuild, cmd.ForceBuild, cmd.ForceDeploy, cmd.VerboseDependencies, configutil.FromFlags(cmd.GlobalFlags), log.GetInstance())
		if err != nil {
			return 0, errors.Errorf("Error deploying dependencies: %v", err)
		}

		// Build image if necessary
		builtImages := make(map[string]string)
		if cmd.SkipBuild == false {
			builtImages, err = build.All(config, generatedConfig.GetActive(), client, cmd.SkipPush, true, cmd.ForceBuild, cmd.BuildSequential, skipBuildIfAlreadyBuilt, log.GetInstance())
			if err != nil {
				if strings.Index(err.Error(), "no space left on device") != -1 {
					return 0, errors.Errorf("Error building image: %v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
				}

				return 0, errors.Errorf("Error building image: %v", err)
			}

			// Save config if an image was built
			if len(builtImages) > 0 {
				err := generated.SaveConfig(generatedConfig)
				if err != nil {
					return 0, errors.Errorf("Error saving generated config: %v", err)
				}
			}
		}

		// Deploy all defined deployments
		if config.Deployments != nil {
			// What deployments should be deployed
			deployments := []string{}
			if cmd.Deployments != "" {
				deployments = strings.Split(cmd.Deployments, ",")
				for index := range deployments {
					deployments[index] = strings.TrimSpace(deployments[index])
				}
			}

			// Deploy all
			err = deploy.All(config, generatedConfig.GetActive(), client, true, cmd.ForceDeploy, builtImages, deployments, log.GetInstance())
			if err != nil {
				return 0, errors.Errorf("Error deploying: %v", err)
			}

			// Save Config
			err = generated.SaveConfig(generatedConfig)
			if err != nil {
				return 0, errors.Errorf("Error saving generated config: %v", err)
			}
		}
	}

	// Start services
	exitCode := 0
	if cmd.ExitAfterDeploy == false {
		var err error

		// Start services
		exitCode, err = cmd.startServices(config, generatedConfig, client, args, log.GetInstance())
		if err != nil {
			// Check if we should reload
			if _, ok := err.(*reloadError); ok {
				// Get the config
				config, err := cmd.loadConfig()
				if err != nil {
					return 0, err
				}

				// Trigger rebuild & redeploy
				return cmd.buildAndDeploy(config, generatedConfig, client, args, false)
			}

			return 0, err
		}
	}

	return exitCode, nil
}

func (cmd *DevCmd) startServices(config *latest.Config, generatedConfig *generated.Config, client *kubectl.Client, args []string, log log.Logger) (int, error) {
	if cmd.Portforwarding {
		portForwarder, err := services.StartPortForwarding(config, generatedConfig, client, log)
		if err != nil {
			return 0, errors.Errorf("Unable to start portforwarding: %v", err)
		}

		defer func() {
			for _, v := range portForwarder {
				v.Close()
			}
		}()
	}

	if cmd.Sync {
		syncConfigs, err := services.StartSync(config, generatedConfig, client, cmd.VerboseSync, log)
		if err != nil {
			return 0, errors.Errorf("Unable to start sync: %v", err)
		}

		defer func() {
			for _, v := range syncConfigs {
				v.Stop(nil)
			}
		}()
	}

	var (
		exitChan        = make(chan error)
		autoReloadPaths = GetPaths(config)
		interactiveMode = config.Dev != nil && config.Dev.Interactive != nil && config.Dev.Interactive.DefaultEnabled != nil && *config.Dev.Interactive.DefaultEnabled == true
	)

	// Start watcher if we have at least one auto reload path and if we should not skip the pipeline
	if cmd.SkipPipeline == false && len(autoReloadPaths) > 0 {
		var once sync.Once
		watcher, err := watch.New(autoReloadPaths, []string{".devspace/"}, func(changed []string, deleted []string) error {
			once.Do(func() {
				if interactiveMode {
					log.Info("Change detected, will reload in 2 seconds")
					time.Sleep(time.Second * 2)
				} else {
					log.Info("Change detected, will reload")
				}

				exitChan <- &reloadError{}
			})

			return nil
		}, log)
		if err != nil {
			return 0, err
		}

		watcher.Start()
		defer watcher.Stop()
	}

	// Run dev.open configs
	if config.Dev.Open != nil && cmd.SkipOpen == false {
		// Skip executing open config next time (e.g. when automatic redeployment is enabled)
		cmd.SkipOpen = true

		for _, openConfig := range config.Dev.Open {
			if openConfig.URL != "" {
				maxWait := 4 * time.Minute
				log.Infof("Opening '%s' as soon as application will be started (timeout: %s)", openConfig.URL, maxWait)

				go func() {
					// Use DiscardLogger as we do not want to print warnings about failed HTTP requests
					err := openURL(openConfig.URL, nil, "", logutil.Discard, maxWait)
					if err != nil {
						// Use warn instead of fatal to prevent exit
						// Do not print warning
						// log.Warn(err)
					}
				}()
			}
		}
	}

	// Check if we should open a terminal
	if interactiveMode {
		var imageSelector []string
		if config.Dev.Interactive.Terminal != nil && config.Dev.Interactive.Terminal.ImageName != "" {
			imageConfigCache := generatedConfig.GetActive().GetImageCache(config.Dev.Interactive.Terminal.ImageName)
			if imageConfigCache.ImageName != "" {
				imageSelector = []string{imageConfigCache.ImageName + ":" + imageConfigCache.Tag}
			}
		} else if len(config.Dev.Interactive.Images) > 0 {
			imageSelector = []string{}
			cache := generatedConfig.GetActive()

			for _, imageConfig := range config.Dev.Interactive.Images {
				imageConfigCache := cache.GetImageCache(imageConfig.Name)
				if imageConfigCache.ImageName != "" {
					imageSelector = append(imageSelector, imageConfigCache.ImageName+":"+imageConfigCache.Tag)
				}
			}
		}

		selectorParameter := &targetselector.SelectorParameter{
			CmdParameter: targetselector.CmdParameter{
				Namespace:   cmd.Namespace,
				Interactive: true,
			},
		}

		if config != nil && config.Dev != nil && config.Dev.Interactive != nil && config.Dev.Interactive.Terminal != nil {
			selectorParameter.ConfigParameter = targetselector.ConfigParameter{
				Namespace:     config.Dev.Interactive.Terminal.Namespace,
				LabelSelector: config.Dev.Interactive.Terminal.LabelSelector,
				ContainerName: config.Dev.Interactive.Terminal.ContainerName,
			}
		}

		return services.StartTerminal(config, client, selectorParameter, args, imageSelector, exitChan, log)
	}

	// Check if we should show logs
	if config.Dev == nil || config.Dev.Logs == nil || config.Dev.Logs.Disabled == nil || *config.Dev.Logs.Disabled == false {
		// Build an image selector
		imageSelector := []string{}
		if config.Dev != nil && config.Dev.Logs != nil && config.Dev.Logs.Images != nil {
			for generatedImageName, imageConfigCache := range generatedConfig.GetActive().Images {
				if imageConfigCache.ImageName != "" {
					// Check that they are also in the real config
					for _, configImageName := range config.Dev.Logs.Images {
						if configImageName == generatedImageName {
							imageSelector = append(imageSelector, imageConfigCache.ImageName+":"+imageConfigCache.Tag)
							break
						}
					}
				}
			}
		} else {
			for generatedImageName, imageConfigCache := range generatedConfig.GetActive().Images {
				if imageConfigCache.ImageName != "" {
					// Check that they are also in the real config
					for configImageName := range config.Images {
						if configImageName == generatedImageName {
							imageSelector = append(imageSelector, imageConfigCache.ImageName+":"+imageConfigCache.Tag)
							break
						}
					}
				}
			}
		}

		// Show last log lines
		tail := int64(50)
		if config.Dev != nil && config.Dev.Logs != nil && config.Dev.Logs.ShowLast != nil {
			tail = int64(*config.Dev.Logs.ShowLast)
		}

		// Log multiple images at once
		err := client.LogMultiple(imageSelector, exitChan, &tail, os.Stdout, log)
		if err != nil {
			// Check if we should reload
			if _, ok := err.(*reloadError); ok {
				return 0, err
			}

			log.Warnf("Couldn't print logs: %v", err)
		}
		log.Warn("Log streaming services has been terminated")
	}

	log.Done("Sync and port-forwarding services are running (Press Ctrl+C to abort services)")
	return 0, <-exitChan
}

func (cmd *DevCmd) validateFlags() error {
	if cmd.SkipBuild && cmd.ForceBuild {
		return errors.New("Flags --skip-build & --force-build cannot be used together")
	}

	return nil
}

// GetPaths retrieves the watch paths from the config object
func GetPaths(config *latest.Config) []string {
	paths := make([]string, 0, 1)

	// Add the deploy manifest paths
	if config.Dev != nil && config.Dev.AutoReload != nil {
		if config.Dev.AutoReload.Deployments != nil && config.Deployments != nil {
			for _, deployName := range config.Dev.AutoReload.Deployments {
				for _, deployConf := range config.Deployments {
					if deployName == deployConf.Name {
						if deployConf.Helm != nil && deployConf.Helm.Chart.Name != "" {
							_, err := os.Stat(deployConf.Helm.Chart.Name)
							if err == nil {
								chartPath := deployConf.Helm.Chart.Name
								if chartPath[len(chartPath)-1] != '/' {
									chartPath += "/"
								}

								paths = append(paths, chartPath+"**")
							}
						} else if deployConf.Kubectl != nil && deployConf.Kubectl.Manifests != nil {
							for _, manifestPath := range deployConf.Kubectl.Manifests {
								paths = append(paths, manifestPath)
							}
						}
					}
				}
			}
		}

		// Add the dockerfile paths
		if config.Dev.AutoReload.Images != nil && config.Images != nil {
			for _, imageName := range config.Dev.AutoReload.Images {
				for imageConfName, imageConf := range config.Images {
					if imageName == imageConfName {
						dockerfilePath := "./Dockerfile"
						if imageConf.Dockerfile != "" {
							dockerfilePath = imageConf.Dockerfile
						}

						paths = append(paths, dockerfilePath)
					}
				}
			}
		}

		// Add the additional paths
		if config.Dev.AutoReload.Paths != nil {
			for _, path := range config.Dev.AutoReload.Paths {
				paths = append(paths, path)
			}
		}
	}

	return paths
}

type reloadError struct {
}

func (r *reloadError) Error() string {
	return ""
}

func (cmd *DevCmd) loadConfig() (*latest.Config, error) {
	configutil.ResetConfig()

	// Load config
	config, err := configutil.GetConfig(configutil.FromFlags(cmd.GlobalFlags))
	if err != nil {
		return nil, err
	}

	// Adjust config for interactive mode
	interactiveModeInConfigEnabled := config.Dev != nil && config.Dev.Interactive != nil && config.Dev.Interactive.DefaultEnabled != nil && *config.Dev.Interactive.DefaultEnabled == true
	if cmd.Interactive || interactiveModeInConfigEnabled {
		if config.Dev == nil {
			config.Dev = &latest.DevConfig{}
		}
		if config.Dev.Interactive == nil {
			config.Dev.Interactive = &latest.InteractiveConfig{}
		}

		images := config.Images
		if config.Dev.Interactive.Images == nil && config.Dev.Interactive.Terminal == nil {
			if config.Images == nil || len(config.Images) == 0 {
				return nil, errors.New("Your configuration does not contain any images to build for interactive mode. If you simply want to start the terminal instead of streaming the logs, run `devspace dev -t`")
			}

			imageNames := make([]string, 0, len(images))
			for k := range images {
				imageNames = append(imageNames, k)
			}

			// If only one image exists, use it, otherwise show image picker
			imageName := ""
			if len(imageNames) == 1 {
				imageName = imageNames[0]
			} else {
				imageName, err = survey.Question(&survey.QuestionOptions{
					Question: "Which image do you want to build using the 'ENTRPOINT [sleep, 999999]' override?",
					Options:  imageNames,
				}, log.GetInstance())
				if err != nil {
					return nil, err
				}
			}

			config.Dev.Interactive.Images = []*latest.InteractiveImageConfig{
				{
					Name: imageName,
				},
			}
		}

		// Set image entrypoints if necessary
		for _, imageConf := range config.Dev.Interactive.Images {
			if imageConf.Entrypoint == nil && imageConf.Cmd == nil {
				imageConf.Entrypoint = []string{"sleep"}
				imageConf.Cmd = []string{"999999999"}
			}
		}

		log.Info("Interactive mode: enable terminal")
		config.Dev.Interactive.DefaultEnabled = ptr.Bool(true)
	} else {
		if config.Dev != nil && config.Dev.Interactive != nil {
			config.Dev.Interactive = nil
		}
	}

	return config, nil
}
