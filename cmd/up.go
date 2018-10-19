package cmd

import (
	"os/exec"
	"strings"

	"github.com/covexo/devspace/pkg/util/stdinutil"

	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/deploy"
	deployHelm "github.com/covexo/devspace/pkg/devspace/deploy/helm"
	deployKubectl "github.com/covexo/devspace/pkg/devspace/deploy/kubectl"
	"github.com/covexo/devspace/pkg/devspace/image"
	"github.com/covexo/devspace/pkg/devspace/services"

	"github.com/covexo/devspace/pkg/util/log"

	"github.com/covexo/devspace/pkg/devspace/registry"

	helmClient "github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	k8sv1beta1 "k8s.io/api/rbac/v1beta1"
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
	container       string
	labelSelector   string
	namespace       string
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

const clusterRoleBindingName = "devspace-users"

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
	cobraCmd.Flags().StringVarP(&cmd.flags.container, "container", "c", cmd.flags.container, "Container name where to open the shell")
	cobraCmd.Flags().BoolVar(&cmd.flags.sync, "sync", cmd.flags.sync, "Enable code synchronization")
	cobraCmd.Flags().BoolVar(&cmd.flags.verboseSync, "verbose-sync", cmd.flags.verboseSync, "When enabled the sync will log every file change")
	cobraCmd.Flags().BoolVar(&cmd.flags.portforwarding, "portforwarding", cmd.flags.portforwarding, "Enable port forwarding")
	cobraCmd.Flags().BoolVarP(&cmd.flags.deploy, "deploy", "d", cmd.flags.deploy, "Force chart deployment")
	cobraCmd.Flags().BoolVar(&cmd.flags.switchContext, "switch-context", cmd.flags.switchContext, "Switch kubectl context to the devspace context")
	cobraCmd.Flags().BoolVar(&cmd.flags.exitAfterDeploy, "exit-after-deploy", cmd.flags.exitAfterDeploy, "Exits the command after building the images and deploying the devspace")
	cobraCmd.Flags().StringVarP(&cmd.flags.namespace, "namespace", "n", "", "Namespace where to select pods")
	cobraCmd.Flags().StringVarP(&cmd.flags.labelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
}

// Run executes the command logic
func (cmd *UpCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()
	var err error

	configExists, _ := configutil.ConfigExists()
	if !configExists {
		initCmd := &InitCmd{
			flags: InitCmdFlagsDefault,
		}

		initCmd.Run(nil, []string{})

		// Ensure that config is initialized correctly
		config := configutil.GetConfig()
		configutil.SetDefaults(config)
	}

	// Create kubectl client
	cmd.kubectl, err = kubectl.NewClientWithContextSwitch(cmd.flags.switchContext)
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	err = cmd.ensureNamespace()
	if err != nil {
		log.Fatalf("Unable to create namespace: %v", err)
	}

	err = cmd.ensureClusterRoleBinding()
	if err != nil {
		log.Fatalf("Unable to create ClusterRoleBinding: %v", err)
	}

	if cmd.flags.initRegistries {
		cmd.initRegistries()
	}

	cmd.buildAndDeploy()

	if cmd.flags.exitAfterDeploy == false {
		cmd.startServices(args)
	}
}

func (cmd *UpCmd) ensureNamespace() error {
	config := configutil.GetConfig()
	defaultNamespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		log.Fatalf("Error getting default namespace: %v", err)
	}

	if defaultNamespace != "default" {
		_, err = cmd.kubectl.CoreV1().Namespaces().Get(defaultNamespace, metav1.GetOptions{})
		if err != nil {
			log.Infof("Create namespace %s", defaultNamespace)

			// Create release namespace
			_, err = cmd.kubectl.CoreV1().Namespaces().Create(&k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultNamespace,
				},
			})
		}
	}

	return err
}

func (cmd *UpCmd) ensureClusterRoleBinding() error {
	if kubectl.IsMinikube() {
		return nil
	}

	_, err := cmd.kubectl.RbacV1beta1().ClusterRoleBindings().Get(clusterRoleBindingName, metav1.GetOptions{})
	if err != nil {
		clusterConfig, _ := kubectl.GetClientConfig(false)
		if clusterConfig.AuthProvider != nil && clusterConfig.AuthProvider.Name == "gcp" {
			createRoleBinding := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want the ClusterRoleBinding '" + clusterRoleBindingName + "' to be created automatically? (yes|no)",
				DefaultValue:           "yes",
				ValidationRegexPattern: "^(yes)|(no)$",
			})

			if *createRoleBinding == "no" {
				log.Fatal("Please create ClusterRoleBinding '" + clusterRoleBindingName + "' manually")
			}
			username := configutil.String("")

			log.StartWait("Checking gcloud account")
			gcloudOutput, gcloudErr := exec.Command("gcloud", "config", "list", "account", "--format", "value(core.account)").Output()
			log.StopWait()

			if gcloudErr == nil {
				gcloudEmail := strings.TrimSuffix(strings.TrimSuffix(string(gcloudOutput), "\r\n"), "\n")

				if gcloudEmail != "" {
					username = &gcloudEmail
				}
			}

			username = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "What is the email address of your Google Cloud account?",
				DefaultValue:           *username,
				ValidationRegexPattern: ".+",
			})

			rolebinding := &k8sv1beta1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterRoleBindingName,
				},
				Subjects: []k8sv1beta1.Subject{
					{
						Kind: "User",
						Name: *username,
					},
				},
				RoleRef: k8sv1beta1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "cluster-admin",
				},
			}

			_, err = cmd.kubectl.RbacV1beta1().ClusterRoleBindings().Create(rolebinding)
			if err != nil {
				return err
			}
		} else {
			cfg := configutil.GetConfig()

			if cfg.Cluster.CloudProvider == nil || *cfg.Cluster.CloudProvider == "" {
				log.Warn("Unable to check permissions: If you run into errors, please create the ClusterRoleBinding '" + clusterRoleBindingName + "' as described here: https://devspace.covexo.com/docs/advanced/rbac.html")
			}
		}
	}

	return nil
}

func (cmd *UpCmd) initRegistries() {
	config := configutil.GetConfig()
	registryMap := *config.Registries

	if config.InternalRegistry != nil && config.InternalRegistry.Deploy != nil && *config.InternalRegistry.Deploy == true {
		registryConf, regConfExists := registryMap["internal"]
		if !regConfExists {
			log.Fatal("Registry config not found for internal registry")
		}

		log.StartWait("Initializing helm client")
		helm, err := helmClient.NewClient(cmd.kubectl, log.GetInstance(), false)
		log.StopWait()
		if err != nil {
			log.Fatalf("Error initializing helm client: %v", err)
		}

		log.StartWait("Initializing internal registry")
		err = registry.InitInternalRegistry(cmd.kubectl, helm, config.InternalRegistry, registryConf)
		log.StopWait()
		if err != nil {
			log.Fatalf("Internal registry error: %v", err)
		}

		err = configutil.SaveConfig()
		if err != nil {
			log.Fatalf("Saving config error: %v", err)
		}

		log.Done("Internal registry started")
	}

	if registryMap != nil {
		defaultNamespace, err := configutil.GetDefaultNamespace(config)
		if err != nil {
			log.Fatalf("Cannot get default namespace: %v", err)
		}

		for registryName, registryConf := range registryMap {
			if registryConf.Auth != nil && registryConf.Auth.Password != nil {
				if config.DevSpace.Deployments != nil {
					for _, deployConfig := range *config.DevSpace.Deployments {
						username := ""
						password := *registryConf.Auth.Password
						email := "noreply@devspace-cloud.com"
						registryURL := ""

						if registryConf.Auth.Username != nil {
							username = *registryConf.Auth.Username
						}
						if registryConf.URL != nil {
							registryURL = *registryConf.URL
						}

						namespace := *deployConfig.Namespace
						if namespace == "" {
							namespace = defaultNamespace
						}

						log.StartWait("Creating image pull secret for registry: " + registryName)
						err := registry.CreatePullSecret(cmd.kubectl, namespace, registryURL, username, password, email)
						log.StopWait()

						if err != nil {
							log.Fatalf("Failed to create pull secret for registry: %v", err)
						}
					}
				}
			}
		}
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
	mustRedeploy := cmd.buildImages(generatedConfig)

	// Save Config
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatalf("Error saving config: %v", err)
	}

	// Deploy all defined deployments
	if config.DevSpace.Deployments != nil {
		for _, deployConfig := range *config.DevSpace.Deployments {
			var deployClient deploy.Interface

			if deployConfig.Kubectl != nil {
				log.Info("Deploying " + *deployConfig.Name + " with kubectl")

				deployClient, err = deployKubectl.New(cmd.kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Fatalf("Error deploying devspace: deployment %s error: %v", *deployConfig.Name, err)
				}
			} else if deployConfig.Helm != nil {
				log.Info("Deploying " + *deployConfig.Name + " with helm")

				deployClient, err = deployHelm.New(cmd.kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Fatalf("Error deploying devspace: deployment %s error: %v", *deployConfig.Name, err)
				}
			} else {
				log.Fatalf("Error deploying devspace: deployment %s has no deployment method", *deployConfig.Name)
			}

			err = deployClient.Deploy(generatedConfig, mustRedeploy || cmd.flags.deploy)
			if err != nil {
				log.Fatalf("Error deploying %s: %v", *deployConfig.Name, err)
			}

			log.Donef("Successfully deployed %s", *deployConfig.Name)
		}

		// Save Config
		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatalf("Error saving config: %v", err)
		}
	}
}

// returns true when one of the images had to be rebuild
func (cmd *UpCmd) buildImages(generatedConfig *generated.Config) bool {
	re := false

	config := configutil.GetConfig()

	for imageName, imageConf := range *config.Images {
		shouldRebuild, err := image.Build(cmd.kubectl, generatedConfig, imageName, imageConf, cmd.flags.build)
		if err != nil {
			log.Fatal(err)
		}

		if shouldRebuild {
			re = true
		}
	}

	return re
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

	services.StartTerminal(cmd.kubectl, cmd.flags.container, cmd.flags.labelSelector, cmd.flags.namespace, args, log.GetInstance())
}
