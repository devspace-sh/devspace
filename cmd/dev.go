package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/watch"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/image"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// DevCmd is a struct that defines a command call for "up"
type DevCmd struct {
	flags *DevCmdFlags
}

// DevCmdFlags are the flags available for the up-command
type DevCmdFlags struct {
	initRegistries  bool
	build           bool
	sync            bool
	terminal        bool
	deploy          bool
	exitAfterDeploy bool
	skipPipeline    bool
	switchContext   bool
	portforwarding  bool
	verboseSync     bool
	selector        string
	container       string
	labelSelector   string
	namespace       string
	config          string
	configOverwrite string
}

// DevFlagsDefault are the default flags for DevCmdFlags
var DevFlagsDefault = &DevCmdFlags{
	initRegistries:  true,
	build:           false,
	sync:            true,
	terminal:        true,
	switchContext:   false,
	exitAfterDeploy: false,
	skipPipeline:    false,
	deploy:          false,
	portforwarding:  true,
	verboseSync:     false,
	container:       "",
	namespace:       "",
	labelSelector:   "",
}

func init() {
	cmd := &DevCmd{
		flags: DevFlagsDefault,
	}

	cobraCmd := &cobra.Command{
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
5. Enters the container shell
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().BoolVar(&cmd.flags.initRegistries, "init-registries", cmd.flags.initRegistries, "Initialize registries (and install internal one)")

	cobraCmd.Flags().BoolVarP(&cmd.flags.build, "force-build", "b", cmd.flags.build, "Forces to build every image")
	cobraCmd.Flags().BoolVarP(&cmd.flags.deploy, "force-deploy", "d", cmd.flags.deploy, "Forces to deploy every deployment")

	cobraCmd.Flags().BoolVarP(&cmd.flags.skipPipeline, "skip-pipeline", "x", cmd.flags.skipPipeline, "Skips build & deployment and only starts sync, portforwarding & terminal")

	cobraCmd.Flags().BoolVar(&cmd.flags.sync, "sync", cmd.flags.sync, "Enable code synchronization")
	cobraCmd.Flags().BoolVar(&cmd.flags.verboseSync, "verbose-sync", cmd.flags.verboseSync, "When enabled the sync will log every file change")

	cobraCmd.Flags().BoolVar(&cmd.flags.portforwarding, "portforwarding", cmd.flags.portforwarding, "Enable port forwarding")

	cobraCmd.Flags().BoolVar(&cmd.flags.terminal, "terminal", cmd.flags.terminal, "Enable terminal (true or false)")
	cobraCmd.Flags().StringVarP(&cmd.flags.selector, "selector", "s", "", "Selector name (in config) to select pods/container for terminal")
	cobraCmd.Flags().StringVarP(&cmd.flags.container, "container", "c", cmd.flags.container, "Container name where to open the shell")
	cobraCmd.Flags().StringVarP(&cmd.flags.labelSelector, "label-selector", "l", "", "Comma separated key=value selector list to use for terminal (e.g. release=test)")
	cobraCmd.Flags().StringVarP(&cmd.flags.namespace, "namespace", "n", "", "Namespace where to select pods for terminal")

	cobraCmd.Flags().BoolVar(&cmd.flags.switchContext, "switch-context", cmd.flags.switchContext, "Switch kubectl context to the DevSpace context")
	cobraCmd.Flags().BoolVar(&cmd.flags.exitAfterDeploy, "exit-after-deploy", cmd.flags.exitAfterDeploy, "Exits the command after building the images and deploying the project")

	cobraCmd.Flags().StringVar(&cmd.flags.config, "config", configutil.ConfigPath, "The DevSpace config file to load (default: '.devspace/config.yaml'")

	var devAlias = &cobra.Command{
		Use:   "up",
		Short: "alias for `devspace dev` (deprecated)",
		Run: func(cobraCmd *cobra.Command, args []string) {
			log.Warn("`devspace up` is deprecated, please use `devspace dev` in future")
			cmd.Run(cobraCmd, args)
		},
	}
	rootCmd.AddCommand(devAlias)
}

// Run executes the command logic
func (cmd *DevCmd) Run(cobraCmd *cobra.Command, args []string) {
	if configutil.ConfigPath != cmd.flags.config {
		configutil.ConfigPath = cmd.flags.config
	}

	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Start file logging
	log.StartFileLogging()

	// Configure cloud provider
	err = cloud.Configure(log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to configure cloud provider: %v", err)
	}

	// Create kubectl client and switch context if specified
	client, err := kubectl.NewClientWithContextSwitch(cmd.flags.switchContext)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	// Create namespace if necessary
	err = kubectl.EnsureDefaultNamespace(client, log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to create namespace: %v", err)
	}

	// Create cluster role binding if necessary
	err = kubectl.EnsureGoogleCloudClusterRoleBinding(client, log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to create ClusterRoleBinding: %v", err)
	}

	// Init image registries
	if cmd.flags.initRegistries {
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			dockerClient = nil
		}

		err = registry.InitRegistries(dockerClient, client, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}
	}

	// Build and deploy images
	err = buildAndDeploy(client, cmd.flags, args)
	if err != nil {
		log.Fatal(err)
	}
}

func buildAndDeploy(client *kubernetes.Clientset, flags *DevCmdFlags, args []string) error {
	config := configutil.GetConfig()

	if flags.skipPipeline == false {
		// Load config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			return fmt.Errorf("Error loading generated.yaml: %v", err)
		}

		// Build image if necessary
		mustRedeploy, err := image.BuildAll(client, generatedConfig, true, flags.build, log.GetInstance())
		if err != nil {
			return fmt.Errorf("Error building image: %v", err)
		}

		// Save config if an image was built
		if mustRedeploy == true {
			err := generated.SaveConfig(generatedConfig)
			if err != nil {
				return fmt.Errorf("Error saving generated config: %v", err)
			}
		}

		// Deploy all defined deployments
		if config.Deployments != nil {
			// Deploy all
			err = deploy.All(client, generatedConfig, true, mustRedeploy || flags.deploy, log.GetInstance())
			if err != nil {
				return fmt.Errorf("Error deploying: %v", err)
			}

			// Save Config
			err = generated.SaveConfig(generatedConfig)
			if err != nil {
				return fmt.Errorf("Error saving generated config: %v", err)
			}
		}
	}

	// Start services
	if flags.exitAfterDeploy == false {
		// Start services
		err := startServices(client, flags, args, log.GetInstance())
		if err != nil {
			// Check if we should reload
			if _, ok := err.(*reloadError); ok {
				// Trigger rebuild & redeploy
				return buildAndDeploy(client, flags, args)
			}

			return err
		}
	}

	return nil
}

func startServices(client *kubernetes.Clientset, flags *DevCmdFlags, args []string, log log.Logger) error {
	if flags.portforwarding {
		portForwarder, err := services.StartPortForwarding(client, log)
		if err != nil {
			return fmt.Errorf("Unable to start portforwarding: %v", err)
		}

		defer func() {
			for _, v := range portForwarder {
				v.Close()
			}
		}()
	}

	if flags.sync {
		syncConfigs, err := services.StartSync(client, flags.verboseSync, log)
		if err != nil {
			return fmt.Errorf("Unable to start sync: %v", err)
		}

		defer func() {
			for _, v := range syncConfigs {
				v.Stop(nil)
			}
		}()
	}

	// Print domain name if we use a cloud provider and space
	config := configutil.GetConfig()
	if config.Cluster != nil && config.Cluster.CloudProvider != nil {
		generatedConfig, _ := generated.LoadConfig()
		if generatedConfig != nil && generatedConfig.Space != nil && generatedConfig.Space.Domain != nil {
			log.Infof("The Space is now reachable via ingress on this URL: https://%s", *generatedConfig.Space.Domain)
		}
	}

	exitChan := make(chan error)
	autoReloadPaths := GetPaths()

	// Start watcher if we have at least one auto reload path and if we should not skip the pipeline
	if flags.skipPipeline == false && len(autoReloadPaths) > 0 {
		var once sync.Once
		watcher, err := watch.New(autoReloadPaths, func(changed []string, deleted []string) error {
			once.Do(func() {
				log.Info("Change detected, will reload in 2 seconds")
				time.Sleep(time.Second * 2)

				exitChan <- &reloadError{}
			})

			return nil
		}, log)
		if err != nil {
			return err
		}

		watcher.Start()
		defer watcher.Stop()
	}

	if flags.terminal && (config.Dev == nil || config.Dev.Terminal == nil || config.Dev.Terminal.Disabled == nil || *config.Dev.Terminal.Disabled == false) {
		return services.StartTerminal(client, flags.selector, flags.container, flags.labelSelector, flags.namespace, false, args, exitChan, log)
	}

	log.Info("Will now try to print the logs of a running pod...")

	// Start attaching to a running pod
	err := services.StartAttach(client, flags.selector, flags.container, flags.labelSelector, flags.namespace, exitChan, log)
	if err != nil {
		// If it's a reload error we return that so we can rebuild & redeploy
		if _, ok := err.(*reloadError); ok {
			return err
		}

		log.Infof("Couldn't print logs of running pod: %v", err)
	}

	log.Done("Services started (Press Ctrl+C to abort port-forwarding and sync)")
	return <-exitChan
}

// GetPaths retrieves the watch paths from the config object
func GetPaths() []string {
	paths := make([]string, 0, 1)
	config := configutil.GetConfig()

	// Add the deploy manifest paths
	if config.Dev != nil && config.Dev.AutoReload != nil {
		if config.Dev.AutoReload.Deployments != nil && config.Deployments != nil {
			for _, deployName := range *config.Dev.AutoReload.Deployments {
				for _, deployConf := range *config.Deployments {
					if *deployName == *deployConf.Name {
						if deployConf.Helm != nil && deployConf.Helm.ChartPath != nil {
							chartPath := *deployConf.Helm.ChartPath
							if chartPath[len(chartPath)-1] != '/' {
								chartPath += "/"
							}

							paths = append(paths, chartPath+"**")
						} else if deployConf.Kubectl != nil && deployConf.Kubectl.Manifests != nil {
							for _, manifestPath := range *deployConf.Kubectl.Manifests {
								paths = append(paths, *manifestPath)
							}
						}
					}
				}
			}
		}

		// Add the dockerfile paths
		if config.Dev.AutoReload.Images != nil && config.Images != nil {
			for _, imageName := range *config.Dev.AutoReload.Images {
				for imageConfName, imageConf := range *config.Images {
					if *imageName == imageConfName {
						dockerfilePath := "./Dockerfile"
						if imageConf.Build != nil && imageConf.Build.DockerfilePath != nil {
							dockerfilePath = *imageConf.Build.DockerfilePath
						}

						paths = append(paths, dockerfilePath)
					}
				}
			}
		}

		// Add the additional paths
		if config.Dev.AutoReload.Paths != nil {
			for _, path := range *config.Dev.AutoReload.Paths {
				paths = append(paths, *path)
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
