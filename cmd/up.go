package cmd

import (
	"fmt"
	"time"

	"github.com/covexo/devspace/pkg/devspace/watch"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/deploy"
	"github.com/covexo/devspace/pkg/devspace/docker"
	"github.com/covexo/devspace/pkg/devspace/image"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/devspace/registry"
	"github.com/covexo/devspace/pkg/devspace/services"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// UpCmd is a struct that defines a command call for "up"
type UpCmd struct {
	flags *UpCmdFlags
}

// UpCmdFlags are the flags available for the up-command
type UpCmdFlags struct {
	tiller          bool
	open            string
	initRegistries  bool
	build           bool
	sync            bool
	terminal        bool
	deploy          bool
	exitAfterDeploy bool
	allyes          bool
	switchContext   bool
	portforwarding  bool
	verboseSync     bool
	service         string
	container       string
	labelSelector   string
	namespace       string
	config          string
	configOverwrite string
}

//UpFlagsDefault are the default flags for UpCmdFlags
var UpFlagsDefault = &UpCmdFlags{
	tiller:          true,
	open:            "cmd",
	initRegistries:  true,
	build:           false,
	sync:            true,
	terminal:        true,
	switchContext:   true,
	exitAfterDeploy: false,
	allyes:          false,
	deploy:          false,
	portforwarding:  true,
	verboseSync:     false,
	container:       "",
	namespace:       "",
	labelSelector:   "",
}

func init() {
	cmd := &UpCmd{
		flags: UpFlagsDefault,
	}

	cobraCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts your DevSpace",
		Long: `
#######################################################
#################### devspace up ######################
#######################################################
Starts and connects your DevSpace:
1. Builds your Docker images (if any Dockerfile has changed)
2. Deploys your application via helm or kubectl
3. Forwards container ports to the local computer
4. Starts the sync client
5. Enters the container shell
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().BoolVar(&cmd.flags.tiller, "tiller", cmd.flags.tiller, "Install/upgrade tiller")
	cobraCmd.Flags().BoolVar(&cmd.flags.initRegistries, "init-registries", cmd.flags.initRegistries, "Initialize registries (and install internal one)")

	cobraCmd.Flags().BoolVarP(&cmd.flags.build, "build", "b", cmd.flags.build, "Force image build")
	cobraCmd.Flags().BoolVarP(&cmd.flags.deploy, "deploy", "d", cmd.flags.deploy, "Force chart deployment")

	cobraCmd.Flags().BoolVar(&cmd.flags.sync, "sync", cmd.flags.sync, "Enable code synchronization")
	cobraCmd.Flags().BoolVar(&cmd.flags.verboseSync, "verbose-sync", cmd.flags.verboseSync, "When enabled the sync will log every file change")

	cobraCmd.Flags().BoolVar(&cmd.flags.portforwarding, "portforwarding", cmd.flags.portforwarding, "Enable port forwarding")

	cobraCmd.Flags().BoolVar(&cmd.flags.terminal, "terminal", cmd.flags.terminal, "Enable terminal")
	cobraCmd.Flags().StringVarP(&cmd.flags.service, "service", "s", "", "Service name (in config) to select pods/container for terminal")
	cobraCmd.Flags().StringVarP(&cmd.flags.container, "container", "c", cmd.flags.container, "Container name where to open the shell")
	cobraCmd.Flags().StringVarP(&cmd.flags.labelSelector, "label-selector", "l", "", "Comma separated key=value selector list to use for terminal (e.g. release=test)")
	cobraCmd.Flags().StringVarP(&cmd.flags.namespace, "namespace", "n", "", "Namespace where to select pods for terminal")

	cobraCmd.Flags().BoolVar(&cmd.flags.switchContext, "switch-context", cmd.flags.switchContext, "Switch kubectl context to the devspace context")
	cobraCmd.Flags().BoolVar(&cmd.flags.exitAfterDeploy, "exit-after-deploy", cmd.flags.exitAfterDeploy, "Exits the command after building the images and deploying the devspace")
	cobraCmd.Flags().BoolVarP(&cmd.flags.allyes, "yes", "y", cmd.flags.allyes, "Answer every questions with the default")

	cobraCmd.Flags().StringVar(&cmd.flags.config, "config", configutil.ConfigPath, "The devspace config file to load (default: '.devspace/config.yaml'")
	cobraCmd.Flags().StringVar(&cmd.flags.configOverwrite, "config-overwrite", configutil.OverwriteConfigPath, "The devspace config overwrite file to load (default: '.devspace/overwrite.yaml'")
}

// Run executes the command logic
func (cmd *UpCmd) Run(cobraCmd *cobra.Command, args []string) {
	if configutil.ConfigPath != cmd.flags.config {
		configutil.ConfigPath = cmd.flags.config

		// Don't use overwrite config if we use a different config
		configutil.OverwriteConfigPath = ""
	}
	if configutil.OverwriteConfigPath != cmd.flags.configOverwrite {
		configutil.OverwriteConfigPath = cmd.flags.configOverwrite
	}

	log.StartFileLogging()
	log.Infof("Loading config %s with overwrite config %s", configutil.ConfigPath, configutil.OverwriteConfigPath)

	configExists, _ := configutil.ConfigExists()
	if !configExists {
		log.Write([]byte("\n"))

		initFlags := &InitCmdFlags{
			reconfigure:      false,
			overwrite:        false,
			skipQuestions:    cmd.flags.allyes,
			templateRepoURL:  "https://github.com/covexo/devspace-templates.git",
			templateRepoPath: "",
			language:         "",
		}
		initCmd := &InitCmd{
			flags: initFlags,
		}
		initCmd.Run(nil, []string{})

		// Ensure that config is initialized correctly
		configutil.SetDefaultsOnce()
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

func buildAndDeploy(client *kubernetes.Clientset, flags *UpCmdFlags, args []string) error {
	config := configutil.GetConfig()

	// Load config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		return fmt.Errorf("Error loading generated.yaml: %v", err)
	}

	// Build image if necessary
	mustRedeploy, err := image.BuildAll(client, generatedConfig, flags.build, log.GetInstance())
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
	if config.DevSpace.Deployments != nil {
		// Deploy all
		err = deploy.All(client, generatedConfig, mustRedeploy || flags.deploy, true, log.GetInstance())
		if err != nil {
			return fmt.Errorf("Error deploying devspace: %v", err)
		}

		// Save Config
		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			return fmt.Errorf("Error saving generated config: %v", err)
		}
	}

	// Start services
	if flags.exitAfterDeploy == false {
		// Start services
		err = startServices(client, flags, args, log.GetInstance())
		if err != nil {
			// Check if we should reload
			if _, ok := err.(*reloadError); ok {
				// Force building & redeploying
				flags.build = true
				flags.deploy = true

				return buildAndDeploy(client, flags, args)
			}

			return err
		}
	}

	return nil
}

func startServices(client *kubernetes.Clientset, flags *UpCmdFlags, args []string, log log.Logger) error {
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

	// Print domain name if we use a cloud provider
	// TODO: Change this
	if cloud.DevSpaceURL != "" {
		log.Infof("Your DevSpace is now reachable via ingress on this URL: http://%s", cloud.DevSpaceURL)
		log.Info("See https://devspace-cloud.com/domain-guide for more information")
	}

	config := configutil.GetConfig()
	exitChan := make(chan error)
	autoReloadPaths := watch.GetPaths()

	// Start watcher if we have at least one auto reload path
	if len(autoReloadPaths) > 0 {
		watcher, err := watch.New(autoReloadPaths, func() error {
			log.Info("Change detected, will reload in 2 seconds")
			time.Sleep(time.Second * 2)

			exitChan <- &reloadError{}
			return nil
		}, log)
		if err != nil {
			return err
		}

		watcher.Start()
	}

	if flags.terminal && (config.DevSpace == nil || config.DevSpace.Terminal == nil || config.DevSpace.Terminal.Disabled == nil || *config.DevSpace.Terminal.Disabled == false) {
		return services.StartTerminal(client, flags.service, flags.container, flags.labelSelector, flags.namespace, args, exitChan, log)
	}

	log.Info("Will now try to attach to a running devspace pod...")

	// Start attaching to a running devspace pod
	err := services.StartAttach(client, flags.service, flags.container, flags.labelSelector, flags.namespace, exitChan, log)
	if err != nil {
		// If it's a reload error we return that so we can rebuild & redeploy
		if _, ok := err.(*reloadError); ok {
			return err
		}

		log.Infof("Couldn't attach to a running devspace pod: %v", err)
	}

	log.Done("Services started (Press Ctrl+C to abort port-forwarding and sync)")
	return <-exitChan
}

type reloadError struct {
}

func (r *reloadError) Error() string {
	return ""
}
