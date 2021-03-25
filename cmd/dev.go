package cmd

import (
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/server"
	"github.com/loft-sh/devspace/pkg/devspace/services"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/analyze"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/deploy"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/devspace/watch"
	"github.com/mgutz/ansi"

	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	latest "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DevCmd is a struct that defines a command call for "up"
type DevCmd struct {
	*flags.GlobalFlags

	SkipPush                bool
	SkipPushLocalKubernetes bool
	AllowCyclicDependencies bool
	VerboseDependencies     bool
	Open                    bool

	ForceBuild          bool
	SkipBuild           bool
	BuildSequential     bool
	MaxConcurrentBuilds int

	ForceDeploy       bool
	Deployments       string
	ForceDependencies bool

	Sync            bool
	ExitAfterDeploy bool
	SkipPipeline    bool
	Portforwarding  bool
	VerboseSync     bool
	PrintSyncLog    bool

	UI     bool
	UIPort int

	Terminal         bool
	WorkingDirectory string
	Interactive      bool

	Wait    bool
	Timeout int

	configLoader loader.ConfigLoader
	log          log.Logger
}

// NewDevCmd creates a new devspace dev command
func NewDevCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &DevCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

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

Open terminal instead of logs:
- Use "devspace dev -t" for opening a terminal
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Print upgrade message if new version available
			upgrade.PrintUpgradeMessage()

			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}

	devCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")
	devCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", false, "Deploys the dependencies verbosely")

	devCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	devCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	devCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	devCmd.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")

	devCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to deploy every deployment")
	devCmd.Flags().StringVar(&cmd.Deployments, "deployments", "", "Only deploy a specifc deployment (You can specify multiple deployments comma-separated")
	devCmd.Flags().BoolVar(&cmd.ForceDependencies, "force-dependencies", true, "Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)")

	devCmd.Flags().BoolVarP(&cmd.SkipPipeline, "skip-pipeline", "x", false, "Skips build & deployment and only starts sync, portforwarding & terminal")
	devCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "Skips image pushing, useful for minikube deployment")
	devCmd.Flags().BoolVar(&cmd.SkipPushLocalKubernetes, "skip-push-local-kube", true, "Skips image pushing, if a local kubernetes environment is detected")

	devCmd.Flags().BoolVar(&cmd.UI, "ui", true, "Start the ui server")
	devCmd.Flags().IntVar(&cmd.UIPort, "ui-port", 0, "The port to use when opening the ui server")
	devCmd.Flags().BoolVar(&cmd.Open, "open", true, "Open defined URLs in the browser, if defined")
	devCmd.Flags().BoolVar(&cmd.Sync, "sync", true, "Enable code synchronization")
	devCmd.Flags().BoolVar(&cmd.VerboseSync, "verbose-sync", false, "When enabled the sync will log every file change")
	devCmd.Flags().BoolVar(&cmd.PrintSyncLog, "print-sync", false, "If enabled will print the sync log to the terminal")

	devCmd.Flags().BoolVar(&cmd.Portforwarding, "portforwarding", true, "Enable port forwarding")

	devCmd.Flags().BoolVar(&cmd.ExitAfterDeploy, "exit-after-deploy", false, "Exits the command after building the images and deploying the project")
	devCmd.Flags().BoolVarP(&cmd.Interactive, "interactive", "i", false, "Enable interactive mode for images (overrides entrypoint with sleep command) and start terminal proxy")
	devCmd.Flags().BoolVarP(&cmd.Terminal, "terminal", "t", false, "Open a terminal instead of showing logs")
	devCmd.Flags().StringVar(&cmd.WorkingDirectory, "workdir", "", "The working directory where to open the terminal or execute the command")

	devCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "If true will wait first for pods to be running or fails after given timeout")
	devCmd.Flags().IntVar(&cmd.Timeout, "timeout", 120, "Timeout until dev should stop waiting and fail")

	return devCmd
}

// Run executes the command logic
func (cmd *DevCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	if cmd.Interactive {
		cmd.log.Warn("Interactive mode flag is deprecated and will be removed in the future. Please take a look at https://devspace.sh/cli/docs/guides/interactive-mode on how to transition to an interactive profile")
	}

	// Set config root
	cmd.log = f.GetLog()
	cmd.configLoader = f.NewConfigLoader(cmd.ConfigPath)
	configOptions := cmd.ToConfigOptions()
	configExists, err := cmd.configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// Start file logging
	log.StartFileLogging()

	// Validate flags
	err = cmd.validateFlags()
	if err != nil {
		return err
	}

	// Load generated config
	generatedConfig, err := cmd.configLoader.LoadGenerated(configOptions)
	if err != nil {
		return errors.Errorf("Error loading generated.yaml: %v", err)
	}
	configOptions.GeneratedConfig = generatedConfig

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	// Create kubectl client and switch context if specified
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}
	configOptions.KubeClient = client

	// Show a warning if necessary
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, true, cmd.log)
	if err != nil {
		return err
	}

	// Clear the dependencies & deployments cache if necessary
	clearCache(generatedConfig, client)

	// Deprecated: Fill DEVSPACE_DOMAIN vars
	err = fillDevSpaceDomainVars(client, generatedConfig)
	if err != nil {
		return err
	}

	// restore vars if wanted
	if cmd.RestoreVars {
		vars, _, err := loader.RestoreVarsFromSecret(client, cmd.VarsSecretName)
		if err != nil {
			return errors.Wrap(err, "restore vars")
		} else if vars != nil {
			generatedConfig.Vars = vars
		}
	}

	// Get the config
	configInterface, err := cmd.loadConfig(configOptions)
	if err != nil {
		return err
	}
	config := configInterface.Config()

	// save vars if wanted
	if cmd.SaveVars {
		err = loader.SaveVarsInSecret(client, generatedConfig.Vars, cmd.VarsSecretName, cmd.log)
		if err != nil {
			return errors.Wrap(err, "save vars")
		}
	}

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "dev", client.CurrentContext(), client.Namespace(), config)
	if err != nil {
		return err
	}

	// Create namespace if necessary
	err = client.EnsureDeployNamespaces(config, cmd.log)
	if err != nil {
		return errors.Errorf("Unable to create namespace: %v", err)
	}

	// Create the image pull secrets and add them to the default service account
	dockerClient, err := f.NewDockerClient(cmd.log)
	if err != nil {
		dockerClient = nil
	}

	registryClient := pullsecrets.NewClient(config, generatedConfig.GetActive(), client, dockerClient, cmd.log)
	err = registryClient.CreatePullSecrets()
	if err != nil {
		cmd.log.Warn(err)
	}

	// Build and deploy images
	exitCode, err := cmd.buildAndDeploy(f, configInterface, configOptions, client, args, true)
	if err != nil {
		return err
	} else if exitCode != 0 {
		return &exit.ReturnCodeError{
			ExitCode: exitCode,
		}
	}

	return nil
}

func (cmd *DevCmd) buildAndDeploy(f factory.Factory, configInterface config.Config, configOptions *loader.ConfigOptions, client kubectl.Client, args []string, skipBuildIfAlreadyBuilt bool) (int, error) {
	var (
		config          = configInterface.Config()
		generatedConfig = configInterface.Generated()
	)

	if cmd.SkipPipeline == false {
		// Create Dependencymanager
		manager, err := f.NewDependencyManager(config, generatedConfig, client, cmd.AllowCyclicDependencies, cmd.ToConfigOptions(), cmd.log)
		if err != nil {
			return 0, errors.Wrap(err, "new manager")
		}

		// Dependencies
		err = manager.DeployAll(dependency.DeployOptions{
			ForceDeployDependencies: cmd.ForceDependencies,
			SkipBuild:               cmd.SkipBuild,
			ForceDeploy:             cmd.ForceDeploy,
			Verbose:                 cmd.VerboseDependencies,

			BuildOptions: build.Options{
				SkipPush:                  cmd.SkipPush,
				SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
				ForceRebuild:              cmd.ForceBuild,
				Sequential:                cmd.BuildSequential,
				MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
				IgnoreContextPathChanges:  skipBuildIfAlreadyBuilt,
			},
		})
		if err != nil {
			return 0, errors.Errorf("error deploying dependencies: %v", err)
		}

		// Build image if necessary
		builtImages := make(map[string]string)
		if cmd.SkipBuild == false {
			builtImages, err = f.NewBuildController(config, generatedConfig.GetActive(), client).Build(&build.Options{
				SkipPush:                  cmd.SkipPush,
				SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
				ForceRebuild:              cmd.ForceBuild,
				Sequential:                cmd.BuildSequential,
				MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
				IgnoreContextPathChanges:  skipBuildIfAlreadyBuilt,
			}, cmd.log)
			if err != nil {
				if strings.Index(err.Error(), "no space left on device") != -1 {
					return 0, errors.Errorf("Error building image: %v\n\n Try running `%s` to free docker daemon space and retry", err, ansi.Color("devspace cleanup images", "white+b"))
				}

				return 0, err
			}

			// Save config if an image was built
			if len(builtImages) > 0 {
				err := cmd.configLoader.SaveGenerated(generatedConfig)
				if err != nil {
					return 0, errors.Errorf("error saving generated config: %v", err)
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
			err = f.NewDeployController(config, generatedConfig.GetActive(), client).Deploy(&deploy.Options{
				IsDev:       true,
				ForceDeploy: cmd.ForceDeploy,
				BuiltImages: builtImages,
				Deployments: deployments,
			}, cmd.log)
			if err != nil {
				return 0, errors.Errorf("error deploying: %v", err)
			}

			// Save Config
			err = cmd.configLoader.SaveGenerated(generatedConfig)
			if err != nil {
				return 0, errors.Errorf("error saving generated config: %v", err)
			}
		}

		// Update last used kube context
		err = updateLastKubeContext(cmd.configLoader, client, generatedConfig)
		if err != nil {
			return 0, errors.Wrap(err, "update last kube context")
		}
	}

	// Wait if necessary
	if cmd.Wait {
		report, err := f.NewAnalyzer(client, f.GetLog()).CreateReport(client.Namespace(), analyze.Options{Wait: true, Patient: true, Timeout: cmd.Timeout, IgnorePodRestarts: true})
		if err != nil {
			return 0, errors.Wrap(err, "analyze")
		}

		if len(report) > 0 {
			return 0, errors.Errorf(analyze.ReportToString(report))
		}
	}

	// Start services
	exitCode := 0
	if cmd.ExitAfterDeploy == false {
		var err error

		// Start services
		exitCode, err = cmd.startServices(f, configInterface, client, args, cmd.log)
		if err != nil {
			// Check if we should reload
			if _, ok := err.(*reloadError); ok {
				// Get the config
				configInterface, err := cmd.loadConfig(configOptions)
				if err != nil {
					return 0, err
				}

				// Trigger rebuild & redeploy
				return cmd.buildAndDeploy(f, configInterface, configOptions, client, args, false)
			}

			return 0, err
		}
	}

	return exitCode, nil
}

func (cmd *DevCmd) startServices(f factory.Factory, configInterface config.Config, client kubectl.Client, args []string, logger log.Logger) (int, error) {
	var (
		config          = configInterface.Config()
		generatedConfig = configInterface.Generated()
		servicesClient  = f.NewServicesClient(config, generatedConfig, client, logger)
		exitChan        = make(chan error)
		autoReloadPaths = GetPaths(config)
		interactiveMode = config.Dev != nil && config.Dev.Interactive != nil && config.Dev.Interactive.DefaultEnabled != nil && *config.Dev.Interactive.DefaultEnabled == true
	)

	if cmd.Portforwarding {
		cmd.Portforwarding = false
		err := servicesClient.StartPortForwarding(nil)
		if err != nil {
			return 0, errors.Errorf("Unable to start portforwarding: %v", err)
		}
		err = servicesClient.StartReversePortForwarding(nil)
		if err != nil {
			return 0, errors.Errorf("Unable to start portforwarding: %v", err)
		}
	}

	// Open UI if configured
	if cmd.UI {
		cmd.UI = false
		logger.StartWait("Starting the ui server...")
		defer logger.StopWait()

		var port *int
		if cmd.UIPort != 0 {
			port = &cmd.UIPort
		}

		// Create server
		uiLogger := log.GetFileLogger("ui")
		server, err := server.NewServer(configInterface, "localhost", false, client.CurrentContext(), client.Namespace(), port, uiLogger)
		if err != nil {
			logger.Warnf("Couldn't start UI server: %v", err)
		} else {
			// Start server
			go func() { server.ListenAndServe() }()

			logger.StopWait()
			logger.WriteString("\n#########################################################\n")
			logger.Infof("DevSpace UI available at: %s", ansi.Color("http://"+server.Server.Addr, "white+b"))
			logger.WriteString("#########################################################\n\n")
		}
	}

	if cmd.Sync {
		cmd.Sync = false
		printSyncLog := cmd.PrintSyncLog
		if interactiveMode == false {
			if config == nil || config.Dev == nil || config.Dev.Logs == nil || config.Dev.Logs.Sync == nil || *config.Dev.Logs.Sync == true {
				printSyncLog = true
			}
		}

		err := servicesClient.StartSync(nil, printSyncLog, cmd.VerboseSync)
		if err != nil {
			return 0, errors.Wrap(err, "start sync")
		}
	}

	// Start watcher if we have at least one auto reload path and if we should not skip the pipeline
	if cmd.SkipPipeline == false && len(autoReloadPaths) > 0 {
		var once sync.Once
		watcher, err := watch.New(autoReloadPaths, []string{".devspace/"}, time.Second, func(changed []string, deleted []string) error {
			path := ""
			if len(changed) > 0 {
				path = changed[0]
			} else if len(deleted) > 0 {
				path = deleted[0]
			}

			once.Do(func() {
				if interactiveMode {
					logger.Infof("Change detected in '%s', will reload in 2 seconds", path)
					time.Sleep(time.Second * 2)
				} else {
					logger.Infof("Change detected in '%s', will reload", path)
				}

				exitChan <- &reloadError{}
			})

			return nil
		}, logger)
		if err != nil {
			return 0, err
		}

		watcher.Start()
		defer watcher.Stop()
	}

	// Run dev.open configs
	if config.Dev.Open != nil && cmd.Open == true {
		// Skip executing open config next time (e.g. when automatic redeployment is enabled)
		cmd.Open = false

		for _, openConfig := range config.Dev.Open {
			if openConfig.URL != "" {
				maxWait := 4 * time.Minute
				logger.Infof("Opening '%s' as soon as application will be started (timeout: %s)", openConfig.URL, maxWait)

				go func(url string) {
					// Use DiscardLogger as we do not want to print warnings about failed HTTP requests
					err := openURL(url, nil, "", log.Discard, maxWait)
					if err != nil {
						// Use warn instead of fatal to prevent exit
						// Do not print warning
						// log.Warn(err)
					}
				}(openConfig.URL)
			}
		}
	}

	return cmd.startOutput(interactiveMode, config, generatedConfig, client, args, servicesClient, exitChan, logger)
}

func (cmd *DevCmd) startOutput(interactiveMode bool, config *latest.Config, generatedConfig *generated.Config, client kubectl.Client, args []string, servicesClient services.Client, exitChan chan error, logger log.Logger) (int, error) {
	// Check if we should open a terminal or stream logs
	if cmd.PrintSyncLog == false {
		if interactiveMode {
			selectorOptions := targetselector.NewDefaultOptions().ApplyCmdParameter("", "", cmd.Namespace, "")
			if config != nil && config.Dev != nil && config.Dev.Interactive != nil && config.Dev.Interactive.Terminal != nil {
				selectorOptions = selectorOptions.ApplyConfigParameter(config.Dev.Interactive.Terminal.LabelSelector, config.Dev.Interactive.Terminal.Namespace, config.Dev.Interactive.Terminal.ContainerName, "")
			}

			var imageSelector []string
			if config.Dev.Interactive.Terminal != nil && config.Dev.Interactive.Terminal.ImageName != "" {
				imageSelector = targetselector.ImageSelectorFromConfig(config.Dev.Interactive.Terminal.ImageName, config, generatedConfig.GetActive())
			} else if len(config.Dev.Interactive.Images) > 0 {
				imageSelector = []string{}
				for _, imageConfig := range config.Dev.Interactive.Images {
					imageSelector = append(imageSelector, targetselector.ImageSelectorFromConfig(imageConfig.Name, config, generatedConfig.GetActive())...)
				}
			}

			selectorOptions.ImageSelector = imageSelector
			return servicesClient.StartTerminal(selectorOptions, args, cmd.WorkingDirectory, exitChan, true)
		} else if config.Dev == nil || config.Dev.Logs == nil || config.Dev.Logs.Disabled == nil || *config.Dev.Logs.Disabled == false {
			// Log multiple images at once
			manager, err := services.NewLogManager(client, config, generatedConfig, exitChan, logger)
			if err != nil {
				return 0, errors.Wrap(err, "starting log manager")
			}

			err = manager.Start()
			if err != nil {
				// Check if we should reload
				if _, ok := err.(*reloadError); ok {
					return 0, err
				}

				logger.Warnf("Couldn't print logs: %v", err)
			}

			logger.WriteString("\n")
			logger.Warn("Log streaming service has been terminated")
		}
		logger.Done("Sync and port-forwarding services are running (Press Ctrl+C to abort services)")
	}

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
						if deployConf.Helm != nil {
							// Watch values files
							for _, p := range deployConf.Helm.ValuesFiles {
								paths = append(paths, p)
							}

							if deployConf.Helm.Chart.Name != "" {
								_, err := os.Stat(deployConf.Helm.Chart.Name)
								if err == nil {
									chartPath := deployConf.Helm.Chart.Name
									if chartPath[len(chartPath)-1] != '/' {
										chartPath += "/"
									}

									paths = append(paths, chartPath+"**")
								}
							}
						} else if deployConf.Kubectl != nil && deployConf.Kubectl.Manifests != nil {
							for _, manifestPath := range deployConf.Kubectl.Manifests {
								s, err := os.Stat(manifestPath)
								if err != nil {
									continue
								}

								if s.IsDir() {
									if manifestPath[len(manifestPath)-1] != '/' {
										manifestPath += "/"
									}

									paths = append(paths, manifestPath+"**")
								} else {
									paths = append(paths, manifestPath)
								}
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

	return removeDuplicates(paths)
}

type reloadError struct {
}

func (r *reloadError) Error() string {
	return ""
}

func (cmd *DevCmd) loadConfig(configOptions *loader.ConfigOptions) (config.Config, error) {
	// Load config
	configInterface, err := cmd.configLoader.Load(configOptions, cmd.log)
	if err != nil {
		return nil, err
	}
	config := configInterface.Config()

	// Adjust config for interactive mode
	interactiveModeInConfigEnabled := config.Dev != nil && config.Dev.Interactive != nil && config.Dev.Interactive.DefaultEnabled != nil && *config.Dev.Interactive.DefaultEnabled == true
	if cmd.Terminal || cmd.Interactive || interactiveModeInConfigEnabled {
		if config.Dev == nil {
			config.Dev = &latest.DevConfig{}
		}
		if config.Dev.Interactive == nil {
			config.Dev.Interactive = &latest.InteractiveConfig{}
		}

		images := config.Images
		if config.Dev.Interactive.Images == nil && config.Dev.Interactive.Terminal == nil {
			if config.Images == nil || len(config.Images) == 0 {
				return nil, errors.New(message.ConfigNoImages)
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
				question := "Which image do you want to build using the 'ENTRPOINT [sleep, 999999]' override?"
				if cmd.Terminal {
					question = "Which image do you want to open a terminal to?"
				}

				imageName, err = cmd.log.Question(&survey.QuestionOptions{
					Question: question,
					Options:  imageNames,
				})
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
			if cmd.Terminal {
				imageConf.Entrypoint = nil
				imageConf.Cmd = nil
			} else if imageConf.Entrypoint == nil && imageConf.Cmd == nil {
				imageConf.Entrypoint = []string{"sleep"}
				imageConf.Cmd = []string{"999999999"}
			}

			for imageConfName, imageOverrideConfig := range config.Images {
				if imageConf.Name == imageConfName {
					if imageConf.Entrypoint != nil {
						imageOverrideConfig.Entrypoint = imageConf.Entrypoint
					}
					if imageConf.Cmd != nil {
						imageOverrideConfig.Cmd = imageConf.Cmd
					}
					break
				}
			}

			if imageConf.Entrypoint != nil && imageConf.Cmd != nil {
				cmd.log.Infof("Override image '%s' entrypoint with %+v and cmd with %+v", ansi.Color(imageConf.Name, "white+b"), imageConf.Entrypoint, imageConf.Cmd)
			} else if imageConf.Entrypoint != nil {
				cmd.log.Infof("Override image '%s' entrypoint with %+v", ansi.Color(imageConf.Name, "white+b"), imageConf.Entrypoint)
			} else if imageConf.Cmd != nil {
				cmd.log.Infof("Override image '%s' cmd with %+v", ansi.Color(imageConf.Name, "white+b"), imageConf.Cmd)
			}
		}

		cmd.log.Info("Interactive mode: enable terminal")
		config.Dev.Interactive.DefaultEnabled = ptr.Bool(true)
	} else {
		if config.Dev != nil && config.Dev.Interactive != nil {
			config.Dev.Interactive = nil
		}
	}

	return configInterface, nil
}

func removeDuplicates(arr []string) []string {
	newArr := []string{}
	for _, v := range arr {
		if contains(newArr, v) == false {
			newArr = append(newArr, v)
		}
	}
	return newArr
}

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func updateLastKubeContext(configLoader loader.ConfigLoader, client kubectl.Client, generatedConfig *generated.Config) error {
	// Update generated if we deploy the application
	if generatedConfig != nil {
		generatedConfig.GetActive().LastContext = &generated.LastContextConfig{
			Context:   client.CurrentContext(),
			Namespace: client.Namespace(),
		}

		err := configLoader.SaveGenerated(generatedConfig)
		if err != nil {
			return errors.Wrap(err, "save generated")
		}
	}

	return nil
}
