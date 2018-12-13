package cmd

import (
	"os"
	"path/filepath"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	helmClient "github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/devspace/registry"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/stdinutil"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ResetCmd holds the needed command information
type ResetCmd struct {
	flags   *ResetCmdFlags
	kubectl *kubernetes.Clientset
}

// ResetCmdFlags holds the possible reset cmd flags
type ResetCmdFlags struct {
	config          string
	configOverwrite string
}

func init() {
	cmd := &ResetCmd{
		flags: &ResetCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "reset",
		Short: "Remove devspace completely from your project",
		Long: `
#######################################################
################### devspace reset ####################
#######################################################
Resets your project by removing all DevSpace related 
data from your project and your cluster, including:
1. DevSpace deployments
2. Docker registry (if deployed)
3. DevSpace config files in .devspace/ (local)

Use the flag --all-data to also remove:
1. Tiller server (if deployed)
2. Helm home (if helm is used)

If you simply want to shutdown your DevSpace, use the 
command: devspace down
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}

	cobraCmd.Flags().StringVar(&cmd.flags.config, "config", configutil.ConfigPath, "The devspace config file to load (default: '.devspace/config.yaml'")
	cobraCmd.Flags().StringVar(&cmd.flags.configOverwrite, "config-overwrite", configutil.OverwriteConfigPath, "The devspace config overwrite file to load (default: '.devspace/overwrite.yaml'")

	rootCmd.AddCommand(cobraCmd)
}

// Run executes the reset command logic
func (cmd *ResetCmd) Run(cobraCmd *cobra.Command, args []string) {
	if configutil.ConfigPath != cmd.flags.config {
		configutil.ConfigPath = cmd.flags.config

		// Don't use overwrite config if we use a different config
		configutil.OverwriteConfigPath = ""
	}
	if configutil.OverwriteConfigPath != cmd.flags.configOverwrite {
		configutil.OverwriteConfigPath = cmd.flags.configOverwrite
	}

	log.Infof("Loading config %s with overwrite config %s", configutil.ConfigPath, configutil.OverwriteConfigPath)
	var err error

	// Create kubectl client
	if cmd.kubectl == nil {
		cmd.kubectl, err = kubectl.NewClient()
		if err != nil {
			log.Failf("Failed to initialize kubectl client: %v", err)
		}
	}

	config := configutil.GetConfig()

	if config.Cluster != nil && config.Cluster.CloudProvider != nil && config.Cluster.Namespace != nil && *config.Cluster.Namespace != "" {
		cmd.deleteCloudDevSpace()
	} else {
		cmd.deleteDevSpaceDeployments()
		cmd.deleteInternalRegistry()
		cmd.deleteTiller()
		cmd.deleteClusterRoleBinding()
	}

	cmd.deleteDeploymentFiles()
	cmd.deleteImageFiles()
	cmd.deleteDevspaceFolder()
}

func (cmd *ResetCmd) deleteCloudDevSpace() {
	config := configutil.GetConfig()
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		log.Failf("Error loading cloud config: %v", err)
		return
	}

	shouldCloudDevSpaceRemoved := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "\n\nShould this DevSpace be deleted from DevSpace Cloud (y/n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)$",
	}) == "y"

	if shouldCloudDevSpaceRemoved {
		// Get selected cloud provider from config
		selectedCloudProvider := *config.Cluster.CloudProvider

		if provider, ok := providerConfig[selectedCloudProvider]; ok {
			err := cloud.DeleteDevSpace(provider, *config.Cluster.Namespace)
			if err != nil {
				log.Failf("Error deleting devspace: %v", err)
			} else {
				log.Donef("Successfully deleted devspace %s", *config.Cluster.Namespace)

				err := cloud.DeleteKubeContext(*config.Cluster.Namespace)
				if err != nil {
					log.Failf("Error deleting kube context: %v", err)
				}
			}
		}
	} else {
		cmd.deleteCloudKubeContext()
	}
}

func (cmd *ResetCmd) deleteCloudKubeContext() {
	config := configutil.GetConfig()
	shouldCloudContextRemoved := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "\n\nShould the cloud kube context be removed (y/n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)$",
	}) == "y"

	if shouldCloudContextRemoved {
		err := cloud.DeleteKubeContext(*config.Cluster.Namespace)
		if err != nil {
			log.Failf("Error deleting kube context: %v", err)
		} else {
			log.Done("Successfully deleted kube context")
		}
	}
}

func (cmd *ResetCmd) deleteDevSpaceDeployments() {
	deleteDevSpace(cmd.kubectl, nil)
}

func (cmd *ResetCmd) deleteInternalRegistry() {
	config := configutil.GetConfig()

	if config.InternalRegistry != nil {
		shouldRegistryRemoved := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "\n\nShould the internal registry be removed? (y/n)",
			DefaultValue:           "y",
			ValidationRegexPattern: "^(y|n)$",
		}) == "y"

		if shouldRegistryRemoved {
			isDeployed := helmClient.IsTillerDeployed(cmd.kubectl)
			if isDeployed == false {
				return
			}

			helm, err := helmClient.NewClient(cmd.kubectl, log.GetInstance(), false)
			if err != nil {
				log.Fatalf("Error creating helm client: %v", err)
			}

			_, err = helm.DeleteRelease(registry.InternalRegistryName, true)
			if err != nil {
				log.Failf("Error deleting internal registry: %v", err)
			} else {
				log.Done("Successfully deleted internal registry")
			}
		}
	}
}

func (cmd *ResetCmd) deleteTiller() {
	config := configutil.GetConfig()

	if config.Tiller != nil {
		shouldRemoveTiller := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "\n\nShould the tiller server be removed? (y/n)",
			DefaultValue:           "y",
			ValidationRegexPattern: "^(y|n)$",
		}) == "y"

		if shouldRemoveTiller {
			log.StartWait("Deleting tiller")
			err := helmClient.DeleteTiller(cmd.kubectl)
			log.StopWait()

			if err != nil {
				log.Failf("Error deleting tiller: %s", err.Error())
			} else {
				log.Done("Successfully deleted tiller server")
			}
		}
	}
}

func (cmd *ResetCmd) deleteDeploymentFiles() {
	config := configutil.GetConfig()

	if config.DevSpace != nil && config.DevSpace.Deployments != nil {
		for _, deployConfig := range *config.DevSpace.Deployments {
			if deployConfig.Helm != nil && deployConfig.Helm.ChartPath != nil {
				absChartPath, err := filepath.Abs(*deployConfig.Helm.ChartPath)

				if err == nil {
					_, err := os.Stat(absChartPath)
					if os.IsNotExist(err) == false {
						deleteChart := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
							Question:               "\n\nShould the Chart (" + *deployConfig.Helm.ChartPath + "/*) be removed? (y/n)",
							DefaultValue:           "y",
							ValidationRegexPattern: "^(y|n)$",
						}) == "y"

						if deleteChart {
							os.RemoveAll(absChartPath)
							log.Donef("Successfully deleted %s", *deployConfig.Helm.ChartPath)
						}
					}
				}
			}
		}
	}
}

func (cmd *ResetCmd) deleteImageFiles() {
	config := configutil.GetConfig()

	for _, imageConfig := range *config.Images {
		dockerfilePath := "Dockerfile"
		if imageConfig.Build != nil && imageConfig.Build.DockerfilePath != nil {
			dockerfilePath = *imageConfig.Build.DockerfilePath
		}

		absDockerfilePath, err := filepath.Abs(dockerfilePath)
		if err != nil {
			continue
		}

		_, err = os.Stat(absDockerfilePath)
		if os.IsNotExist(err) == false {
			deleteDockerfile := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "\n\nShould " + dockerfilePath + " be removed? (y/n)",
				DefaultValue:           "y",
				ValidationRegexPattern: "^(y|n)$",
			}) == "y"

			if deleteDockerfile {
				os.Remove(absDockerfilePath)
				log.Donef("Successfully deleted %s", absDockerfilePath)
			}
		}

		contextPath := "."
		if imageConfig.Build != nil && imageConfig.Build.ContextPath != nil {
			contextPath = *imageConfig.Build.ContextPath
		}

		absContextPath, err := filepath.Abs(contextPath)
		if err != nil {
			continue
		}

		absDockerIgnorePath := filepath.Join(absContextPath, ".dockerignore")
		_, err = os.Stat(absDockerIgnorePath)
		if os.IsNotExist(err) == false {
			deleteDockerIgnore := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "\n\nShould " + absDockerIgnorePath + " be removed? (y/n)",
				DefaultValue:           "y",
				ValidationRegexPattern: "^(y|n)$",
			}) == "y"

			if deleteDockerIgnore {
				os.Remove(absDockerIgnorePath)
				log.Donef("Successfully deleted %s", absDockerIgnorePath)
			}
		}
	}
}

func (cmd *ResetCmd) deleteClusterRoleBinding() {
	clusterRoleBindingName := kubectl.ClusterRoleBindingName
	_, err := cmd.kubectl.RbacV1beta1().ClusterRoleBindings().Get(clusterRoleBindingName, metav1.GetOptions{})
	if err == nil {
		deleteRoleBinding := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "\n\nShould the ClusterRoleBinding '" + clusterRoleBindingName + "' be removed? (y/n)",
			DefaultValue:           "y",
			ValidationRegexPattern: "^(y|n)$",
		}) == "y"

		if deleteRoleBinding {
			log.StartWait("Deleting cluster role bindings")
			err = cmd.kubectl.RbacV1beta1().ClusterRoleBindings().Delete(clusterRoleBindingName, &metav1.DeleteOptions{})
			log.StopWait()

			if err != nil {
				log.Failf("Failed to remove ClusterRoleBinding: %v", err)
			} else {
				log.Done("Successfully deleted ClusterRoleBinding '" + clusterRoleBindingName + "'")
			}
		}
	}
}

func (cmd *ResetCmd) deleteDevspaceFolder() {
	deleteDevspaceFolder := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "\n\nShould the .devspace folder be removed? (y/n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)$",
	}) == "y"

	if deleteDevspaceFolder {
		os.RemoveAll(".devspace")
		log.Done("Successfully deleted .devspace folder")
	}
}
