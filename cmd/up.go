package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/deploy"
	"github.com/covexo/devspace/pkg/devspace/image"
	"github.com/covexo/devspace/pkg/devspace/services"

	"github.com/covexo/devspace/pkg/util/log"

	"github.com/covexo/devspace/pkg/devspace/registry"

	"github.com/covexo/devspace/pkg/devspace/kubectl"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// UpCmd is a struct that defines a command call for "up"
type UpCmd struct {
	flags   *UpCmdFlags
	kubectl *kubernetes.Clientset
}

// UpCmdFlags are the flags available for the up-command
type UpCmdFlags struct {
	tiller          bool
	open            string
	initRegistries  bool
	build           bool
	sync            bool
	deploy          bool
	exitAfterDeploy bool
	switchContext   bool
	portforwarding  bool
	verboseSync     bool
	service         string
	container       string
	labelSelector   string
	namespace       string
	config          string
}

//UpFlagsDefault are the default flags for UpCmdFlags
var UpFlagsDefault = &UpCmdFlags{
	tiller:          true,
	open:            "cmd",
	initRegistries:  true,
	build:           false,
	sync:            true,
	switchContext:   false,
	exitAfterDeploy: false,
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
	cobraCmd.Flags().BoolVar(&cmd.flags.sync, "sync", cmd.flags.sync, "Enable code synchronization")
	cobraCmd.Flags().BoolVar(&cmd.flags.verboseSync, "verbose-sync", cmd.flags.verboseSync, "When enabled the sync will log every file change")
	cobraCmd.Flags().BoolVar(&cmd.flags.portforwarding, "portforwarding", cmd.flags.portforwarding, "Enable port forwarding")
	cobraCmd.Flags().BoolVarP(&cmd.flags.deploy, "deploy", "d", cmd.flags.deploy, "Force chart deployment")
	cobraCmd.Flags().BoolVar(&cmd.flags.switchContext, "switch-context", cmd.flags.switchContext, "Switch kubectl context to the devspace context")
	cobraCmd.Flags().BoolVar(&cmd.flags.exitAfterDeploy, "exit-after-deploy", cmd.flags.exitAfterDeploy, "Exits the command after building the images and deploying the devspace")
	cobraCmd.Flags().StringVarP(&cmd.flags.service, "service", "s", "", "Service name (in config) to select pods/container for terminal")
	cobraCmd.Flags().StringVarP(&cmd.flags.container, "container", "c", cmd.flags.container, "Container name where to open the shell")
	cobraCmd.Flags().StringVarP(&cmd.flags.labelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	cobraCmd.Flags().StringVarP(&cmd.flags.namespace, "namespace", "n", "", "Namespace where to select pods")
	cobraCmd.Flags().StringVar(&cmd.flags.config, "config", configutil.ConfigPath, "The devspace config file to load (default: '.devspace/config.yaml'")
}

// Run executes the command logic
func (cmd *UpCmd) Run(cobraCmd *cobra.Command, args []string) {
	if configutil.ConfigPath != cmd.flags.config {
		configutil.ConfigPath = cmd.flags.config

		// Don't use overwrite config if we use a different config
		configutil.OverwriteConfigPath = ""
	}

	log.StartFileLogging()
	var err error

	configExists, _ := configutil.ConfigExists()
	if !configExists {
		initCmd := &InitCmd{
			flags: InitCmdFlagsDefault,
		}

		initCmd.Run(nil, []string{})

		// Ensure that config is initialized correctly
		configutil.SetDefaultsOnce()
	}

	// Create kubectl client
	cmd.kubectl, err = kubectl.NewClientWithContextSwitch(cmd.flags.switchContext)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	// Create namespace if necessary
	err = kubectl.EnsureDefaultNamespace(cmd.kubectl, log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to create namespace: %v", err)
	}

	// Create cluster role binding if necessary
	err = kubectl.EnsureGoogleCloudClusterRoleBinding(cmd.kubectl, log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to create ClusterRoleBinding: %v", err)
	}

	// Init image registries
	if cmd.flags.initRegistries {
		err = registry.InitRegistries(cmd.kubectl, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}
	}

	// Build and deploy images
	cmd.buildAndDeploy()

	if cmd.flags.exitAfterDeploy == false {
		// Start services
		cmd.startServices(args)
	}
}

func (cmd *UpCmd) buildAndDeploy() {
	config := configutil.GetConfig()

	// Load config
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading generated.yaml: %v", err)
	}

	// Build image if necessary
	mustRedeploy, err := image.BuildAll(cmd.kubectl, generatedConfig, cmd.flags.build, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Save config if an image was built
	if mustRedeploy == true {
		err := generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatalf("Error saving config: %v", err)
		}
	}

	// Deploy all defined deployments
	if config.DevSpace.Deployments != nil {
		// Deploy all
		err = deploy.All(cmd.kubectl, generatedConfig, mustRedeploy || cmd.flags.deploy, true, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}

		// Save Config
		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatalf("Error saving config: %v", err)
		}
	}
}

func (cmd *UpCmd) startServices(args []string) {
	if cmd.flags.portforwarding {
		err := services.StartPortForwarding(cmd.kubectl, log.GetInstance())
		if err != nil {
			log.Fatalf("Unable to start portforwarding: %v", err)
		}
	}

	if cmd.flags.sync {
		syncConfigs, err := services.StartSync(cmd.kubectl, cmd.flags.verboseSync, log.GetInstance())
		if err != nil {
			log.Fatalf("Unable to start sync: %v", err)
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
		log.Infof("Your devspace is reachable via ingress on this url http://%s", cloud.DevSpaceURL)
		log.Info("See https://devspace-cloud.com/domain-guide for more information")
	}

	services.StartTerminal(cmd.kubectl, cmd.flags.service, cmd.flags.container, cmd.flags.labelSelector, cmd.flags.namespace, args, log.GetInstance())
}
