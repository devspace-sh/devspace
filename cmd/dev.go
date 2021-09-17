package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/legacy"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/util"
	"github.com/loft-sh/devspace/pkg/devspace/dev"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"github.com/loft-sh/devspace/pkg/util/interrupt"
	"github.com/loft-sh/devspace/pkg/util/survey"

	"github.com/loft-sh/devspace/pkg/devspace/plugin"
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
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DevCmd is a struct that defines a command call for "up"
type DevCmd struct {
	*flags.GlobalFlags

	SkipPush                bool
	SkipPushLocalKubernetes bool
	VerboseDependencies     bool
	Open                    bool

	Dependency     []string
	SkipDependency []string

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

	Terminal          bool
	TerminalReconnect bool
	WorkingDirectory  string
	Interactive       bool

	Wait    bool
	Timeout int

	configLoader loader.ConfigLoader
	log          log.Logger

	// used for testing to allow interruption
	Interrupt chan error
	Stdout    io.Writer
	Stderr    io.Writer
	Stdin     io.Reader
}

// NewDevCmd creates a new devspace dev command
func NewDevCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f, args)
		},
	}

	devCmd.Flags().StringSliceVar(&cmd.SkipDependency, "skip-dependency", []string{}, "Skips the following dependencies for deployment")
	devCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Deploys only the specified named dependencies")
	devCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", true, "Deploys the dependencies verbosely")
	devCmd.Flags().BoolVar(&cmd.ForceDependencies, "force-dependencies", true, "Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)")

	devCmd.Flags().BoolVarP(&cmd.ForceBuild, "force-build", "b", false, "Forces to build every image")
	devCmd.Flags().BoolVar(&cmd.SkipBuild, "skip-build", false, "Skips building of images")
	devCmd.Flags().BoolVar(&cmd.BuildSequential, "build-sequential", false, "Builds the images one after another instead of in parallel")
	devCmd.Flags().IntVar(&cmd.MaxConcurrentBuilds, "max-concurrent-builds", 0, "The maximum number of image builds built in parallel (0 for infinite)")

	devCmd.Flags().BoolVarP(&cmd.ForceDeploy, "force-deploy", "d", false, "Forces to deploy every deployment")
	devCmd.Flags().StringVar(&cmd.Deployments, "deployments", "", "Only deploy a specifc deployment (You can specify multiple deployments comma-separated")

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
	devCmd.Flags().BoolVarP(&cmd.Interactive, "interactive", "i", false, "DEPRECATED: DO NOT USE ANYMORE")
	devCmd.Flags().BoolVarP(&cmd.Terminal, "terminal", "t", false, "Open a terminal instead of showing logs")
	devCmd.Flags().BoolVar(&cmd.TerminalReconnect, "terminal-reconnect", true, "Will try to reconnect the terminal if an unexpected exit code was encountered")
	devCmd.Flags().StringVar(&cmd.WorkingDirectory, "workdir", "", "The working directory where to open the terminal or execute the command")

	devCmd.Flags().BoolVar(&cmd.Wait, "wait", false, "If true will wait first for pods to be running or fails after given timeout")
	devCmd.Flags().IntVar(&cmd.Timeout, "timeout", 120, "Timeout until dev should stop waiting and fail")

	return devCmd
}

// Run executes the command logic
func (cmd *DevCmd) Run(f factory.Factory, args []string) error {
	if cmd.Interactive {
		cmd.log.Warn("Interactive mode flag is deprecated and will be removed in the future. Please take a look at https://devspace.sh/cli/docs/guides/interactive-mode on how to transition to an interactive profile")
	}

	// Set config root
	cmd.log = f.GetLog()
	cmd.configLoader = f.NewConfigLoader(cmd.ConfigPath)
	configOptions := cmd.ToConfigOptions(cmd.log)
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

	// Get the config
	configInterface, err := cmd.loadConfig(configOptions)
	if err != nil {
		return err
	}

	return runWithHooks("devCommand", client, configInterface, cmd.log, func() error {
		// Create namespace if necessary
		err = client.EnsureDeployNamespaces(configInterface.Config(), cmd.log)
		if err != nil {
			return errors.Errorf("Unable to create namespace: %v", err)
		}

		// Create the image pull secrets and add them to the default service account
		dockerClient, err := f.NewDockerClient(cmd.log)
		if err != nil {
			dockerClient = nil
		}

		// Execute plugin hook
		err = hook.ExecuteHooks(client, nil, nil, nil, nil, "dev")
		if err != nil {
			return err
		}

		// Build and deploy images
		exitCode, err := cmd.buildAndDeploy(f, configInterface, configOptions, client, dockerClient, args)
		if err != nil {
			return err
		} else if exitCode != 0 {
			return &exit.ReturnCodeError{
				ExitCode: exitCode,
			}
		}

		return nil
	})
}

func runWithHooks(command string, client kubectl.Client, configInterface config.Config, logger log.Logger, fn func() error) (err error) {
	err = hook.ExecuteHooks(client, configInterface, nil, nil, logger, command+":before:execute")
	if err != nil {
		return err
	}

	defer func() {
		if logger == nil {
			log.GetInstance().StopWait()
		} else {
			logger.StopWait()
		}

		if err != nil {
			hook.LogExecuteHooks(client, configInterface, nil, map[string]interface{}{"error": err}, logger, command+":after:execute", command+":error")
		} else {
			err = hook.ExecuteHooks(client, configInterface, nil, nil, logger, command+":after:execute")
		}
	}()

	return interrupt.Global.Run(fn, func() {
		if logger == nil {
			log.GetInstance().StopWait()
		} else {
			logger.StopWait()
		}

		hook.LogExecuteHooks(client, configInterface, nil, nil, logger, command+":interrupt")
	})
}

func (cmd *DevCmd) buildAndDeploy(f factory.Factory, configInterface config.Config, configOptions *loader.ConfigOptions, client kubectl.Client, dockerClient docker.Client, args []string) (int, error) {
	var (
		err             error
		config          = configInterface.Config()
		generatedConfig = configInterface.Generated()
		dependencies    = []types.Dependency{}
	)

	// execute plugin hook
	err = hook.ExecuteHooks(client, configInterface, dependencies, nil, cmd.log, "dev.beforePipeline", "devCommand:before:runPipeline")
	if err != nil {
		return 0, err
	}

	if !cmd.SkipPipeline {
		pluginErr := hook.ExecuteHooks(client, configInterface, dependencies, nil, cmd.log, "dev.beforeDependencies", "devCommand:before:deployDependencies")
		if pluginErr != nil {
			return 0, pluginErr
		}

		// Dependencies
		dependencies, err = f.NewDependencyManager(configInterface, client, cmd.ToConfigOptions(cmd.log), cmd.log).DeployAll(dependency.DeployOptions{
			ForceDeployDependencies: cmd.ForceDependencies,
			SkipBuild:               cmd.SkipBuild,
			ForceDeploy:             cmd.ForceDeploy,
			Verbose:                 cmd.VerboseDependencies,
			Dependencies:            cmd.Dependency,
			SkipDependencies:        cmd.SkipDependency,

			BuildOptions: build.Options{
				SkipPush:                  cmd.SkipPush,
				SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
				ForceRebuild:              cmd.ForceBuild,
				Sequential:                cmd.BuildSequential,
				MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
			},
		})
		if err != nil {
			return 0, errors.Errorf("error deploying dependencies: %v", err)
		}

		pluginErr = hook.ExecuteHooks(client, configInterface, dependencies, nil, cmd.log, "dev.afterDependencies", "devCommand:after:deployDependencies")
		if pluginErr != nil {
			return 0, pluginErr
		}

		// Create Pull Secrets
		err = pullsecrets.NewClient(configInterface, dependencies, client, dockerClient, cmd.log).CreatePullSecrets()
		if err != nil {
			cmd.log.Warn(err)
		}

		// Only execute pipeline if we are not focused on a dependency
		if len(cmd.Dependency) == 0 {
			pluginErr = hook.ExecuteHooks(client, configInterface, dependencies, nil, cmd.log, "dev.beforeBuild", "devCommand:before:build")
			if pluginErr != nil {
				return 0, pluginErr
			}

			// Build image if necessary
			builtImages := make(map[string]string)
			if !cmd.SkipBuild {
				builtImages, err = f.NewBuildController(configInterface, dependencies, client).Build(&build.Options{
					SkipPush:                  cmd.SkipPush,
					SkipPushOnLocalKubernetes: cmd.SkipPushLocalKubernetes,
					ForceRebuild:              cmd.ForceBuild,
					Sequential:                cmd.BuildSequential,
					MaxConcurrentBuilds:       cmd.MaxConcurrentBuilds,
				}, cmd.log)
				if err != nil {
					if strings.Contains(err.Error(), "no space left on device") {
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

			pluginErr = hook.ExecuteHooks(client, configInterface, dependencies, nil, cmd.log, "dev.afterBuild", "devCommand:after:build", "dev.beforeDeploy", "devCommand:before:deploy")
			if pluginErr != nil {
				return 0, pluginErr
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
				err = f.NewDeployController(configInterface, dependencies, client).Deploy(&deploy.Options{
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

			pluginErr = hook.ExecuteHooks(client, configInterface, dependencies, nil, cmd.log, "dev.afterDeploy", "devCommand:after:deploy")
			if pluginErr != nil {
				return 0, pluginErr
			}
		}

		// Update last used kube context
		err = updateLastKubeContext(cmd.configLoader, client, generatedConfig)
		if err != nil {
			return 0, errors.Wrap(err, "update last kube context")
		}
	}

	pluginErr := hook.ExecuteHooks(client, configInterface, dependencies, nil, cmd.log, "dev.afterPipeline", "devCommand:after:runPipeline")
	if pluginErr != nil {
		return 0, pluginErr
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
	if !cmd.ExitAfterDeploy {
		var err error

		// Start services
		exitCode, err = cmd.startServices(f, configInterface, client, args, dependencies, cmd.log)
		if err != nil {
			// Check if we should reload
			if _, ok := err.(*services.InterruptError); ok {
				// Get the config
				configInterface, err := cmd.loadConfig(configOptions)
				if err != nil {
					return 0, err
				}

				// Trigger rebuild & redeploy
				return cmd.buildAndDeploy(f, configInterface, configOptions, client, dockerClient, args)
			}

			return 0, err
		}
	}

	return exitCode, nil
}

func (cmd *DevCmd) startServices(f factory.Factory, configInterface config.Config, client kubectl.Client, args []string, dependencies []types.Dependency, logger log.Logger) (int, error) {
	var (
		config          = configInterface.Config()
		servicesClient  = f.NewServicesClient(configInterface, dependencies, client, logger)
		exitChan        = make(chan error)
		autoReloadPaths = GetPaths(config)
		useTerminal     = config.Dev.Terminal != nil && !config.Dev.Terminal.Disabled
	)
	if cmd.Interrupt != nil {
		exitChan = cmd.Interrupt
	}

	// replace pods
	err := dev.ReplacePods(servicesClient)
	if err != nil {
		return 0, err
	}

	// Open UI if configured
	if cmd.UI {
		cmd.UI = false

		err := dev.UI(servicesClient, cmd.UIPort)
		if err != nil {
			return 0, err
		}
	}

	if cmd.Portforwarding {
		cmd.Portforwarding = false

		err := dev.PortForwarding(servicesClient, cmd.Interrupt)
		if err != nil {
			return 0, err
		}
	}

	if cmd.Sync {
		cmd.Sync = false
		printSyncLog := cmd.PrintSyncLog
		if !useTerminal && (config.Dev.Logs == nil || config.Dev.Logs.Sync == nil || *config.Dev.Logs.Sync) {
			printSyncLog = true
		}

		err := dev.Sync(servicesClient, cmd.Interrupt, printSyncLog, cmd.VerboseSync)
		if err != nil {
			return 0, err
		}
	}

	// Start watcher if we have at least one auto reload path and if we should not skip the pipeline
	if !cmd.SkipPipeline && len(autoReloadPaths) > 0 {
		var once sync.Once
		watcher, err := watch.New(autoReloadPaths, []string{".devspace/"}, time.Second, func(changed []string, deleted []string) error {
			path := ""
			if len(changed) > 0 {
				path = changed[0]
			} else if len(deleted) > 0 {
				path = deleted[0]
			}

			once.Do(func() {
				if useTerminal {
					logger.Infof("Change detected in '%s', will reload in 2 seconds", path)
					time.Sleep(time.Second * 2)
				} else {
					logger.Infof("Change detected in '%s', will reload", path)
				}

				exitChan <- &services.InterruptError{}
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
	if config != nil && config.Dev.Open != nil && cmd.Open {
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
						_ = err // just to avoid empty branch (SA9003) lint error
					}
				}(openConfig.URL)
			}
		}
	}

	return cmd.startOutput(configInterface, dependencies, client, args, servicesClient, exitChan, logger)
}

func (cmd *DevCmd) startOutput(configInterface config.Config, dependencies []types.Dependency, client kubectl.Client, args []string, servicesClient services.Client, exitChan chan error, logger log.Logger) (int, error) {
	if configInterface == nil {
		return 0, fmt.Errorf("config is nil")
	}

	// get config
	config := configInterface.Config()

	// Check if we should open a terminal or stream logs
	if !cmd.PrintSyncLog {
		if config.Dev.Terminal != nil && !config.Dev.Terminal.Disabled {
			pluginErr := hook.ExecuteHooks(client, configInterface, dependencies, nil, cmd.log, "devCommand:before:openTerminal")
			if pluginErr != nil {
				return 0, pluginErr
			}

			selectorOptions := targetselector.NewDefaultOptions().ApplyCmdParameter("", "", cmd.Namespace, "")
			if config.Dev.Terminal != nil {
				selectorOptions = selectorOptions.ApplyConfigParameter(config.Dev.Terminal.LabelSelector, config.Dev.Terminal.Namespace, config.Dev.Terminal.ContainerName, "")
			}

			var imageSelectors []imageselector.ImageSelector
			if config.Dev.Terminal != nil && config.Dev.Terminal.ImageSelector != "" {
				imageSelector, err := util.ResolveImageAsImageSelector(config.Dev.Terminal.ImageSelector, configInterface, dependencies)
				if err != nil {
					return 0, err
				}

				imageSelectors = append(imageSelectors, *imageSelector)
			}

			selectorOptions.ImageSelector = imageSelectors
			stdout, stderr, stdin := defaultStdStreams(cmd.Stdout, cmd.Stderr, cmd.Stdin)
			code, err := servicesClient.StartTerminal(selectorOptions, args, cmd.WorkingDirectory, exitChan, true, cmd.TerminalReconnect, stdout, stderr, stdin)
			if code != 0 {
				cmd.log.Warnf("Command terminated with exit code %d", code)
			}

			return code, err
		} else if config.Dev.Logs == nil || config.Dev.Logs.Disabled == nil || !*config.Dev.Logs.Disabled {
			pluginErr := hook.ExecuteHooks(client, configInterface, dependencies, nil, cmd.log, "devCommand:before:streamLogs")
			if pluginErr != nil {
				return 0, pluginErr
			}

			// Log multiple images at once
			manager, err := services.NewLogManager(client, configInterface, dependencies, exitChan, logger)
			if err != nil {
				return 0, errors.Wrap(err, "starting log manager")
			}

			err = manager.Start()
			if err != nil {
				// Check if we should reload
				if _, ok := err.(*services.InterruptError); ok {
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
	if config.Dev.AutoReload != nil {
		if config.Dev.AutoReload.Deployments != nil && config.Deployments != nil {
			for _, deployName := range config.Dev.AutoReload.Deployments {
				for _, deployConf := range config.Deployments {
					if deployName == deployConf.Name {
						if deployConf.Helm != nil {
							// Watch values files
							paths = append(paths, deployConf.Helm.ValuesFiles...)

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
			paths = append(paths, config.Dev.AutoReload.Paths...)
		}
	}

	return removeDuplicates(paths)
}

func (cmd *DevCmd) loadConfig(configOptions *loader.ConfigOptions) (config.Config, error) {
	// Load config
	configInterface, err := cmd.configLoader.Load(configOptions, cmd.log)
	if err != nil {
		return nil, err
	}

	// apply legacy interactive mode
	wasInteractive, err := legacy.LegacyInteractiveMode(configInterface.Config(), cmd.Interactive, cmd.Terminal, cmd.log)
	if err != nil {
		return nil, err
	} else if wasInteractive {
		return configInterface, nil
	}

	// check if terminal is enabled
	c := configInterface.Config()

	if cmd.Terminal || (c.Dev.Terminal != nil && !c.Dev.Terminal.Disabled) {
		if c.Dev.Terminal == nil || (c.Dev.Terminal.ImageSelector == "" && len(c.Dev.Terminal.LabelSelector) == 0) {
			imageNames := make([]string, 0, len(c.Images))
			for k := range c.Images {
				imageNames = append(imageNames, k)
			}

			// if only one image exists, use it, otherwise show image picker
			imageName := ""
			if len(imageNames) == 1 {
				imageName = imageNames[0]
			} else {
				imageName, err = cmd.log.Question(&survey.QuestionOptions{
					Question: "Which image do you want to open a terminal to?",
					Options:  imageNames,
				})
				if err != nil {
					return nil, err
				}
			}

			c.Dev.Terminal = &latest.Terminal{
				ImageSelector: fmt.Sprintf("image(%s):tag(%s)", imageName, imageName),
			}
		} else {
			c.Dev.Terminal.Disabled = false
		}
	}

	return configInterface, nil
}

func defaultStdStreams(stdout io.Writer, stderr io.Writer, stdin io.Reader) (io.Writer, io.Writer, io.Reader) {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if stdin == nil {
		stdin = os.Stdin
	}
	return stdout, stderr, stdin
}

func removeDuplicates(arr []string) []string {
	newArr := []string{}
	for _, v := range arr {
		if !contains(newArr, v) {
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
