package cmd

import (
	"os"
	"path/filepath"

	deploy "github.com/devspace-cloud/devspace/pkg/devspace/deploy/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"

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
type ResetCmdFlags struct{}

func init() {
	cmd := &ResetCmd{
		flags: &ResetCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "reset",
		Short: "Remove DevSpace completely from your project",
		Long: `
#######################################################
################### devspace reset ####################
#######################################################
Resets your project by removing all DevSpace related 
data from your project and your cluster, including:
1. DevSpace deployments
3. DevSpace config files in .devspace/ (local)

Use the flag --all-data to also remove:
2. Helm home (if helm is used)

If you simply want to shutdown your DevSpace, use the 
command: devspace down
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}

	rootCmd.AddCommand(cobraCmd)
}

// Run executes the reset command logic
func (cmd *ResetCmd) Run(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	// Create kubectl client
	if cmd.kubectl == nil {
		cmd.kubectl, err = kubectl.NewClient()
		if err != nil {
			log.Failf("Failed to initialize kubectl client: %v", err)
		}
	}

	cmd.deleteDevSpaceDeployments()
	cmd.deleteClusterRoleBinding()
	cmd.deleteImageFiles()
	cmd.deleteDevspaceFolder()
}

func (cmd *ResetCmd) deleteDevSpaceDeployments() {
	deploy.PurgeDeployments(cmd.kubectl, nil)
}

func (cmd *ResetCmd) deleteImageFiles() {
	config := configutil.GetConfig()

	for _, imageConfig := range *config.Images {
		dockerfilePath := "Dockerfile"
		if imageConfig.Build != nil && imageConfig.Build.Dockerfile != nil {
			dockerfilePath = *imageConfig.Build.Dockerfile
		}

		absDockerfilePath, err := filepath.Abs(dockerfilePath)
		if err != nil {
			continue
		}

		_, err = os.Stat(absDockerfilePath)
		if os.IsNotExist(err) == false {
			deleteDockerfile := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:     "Should " + dockerfilePath + " be removed?",
				DefaultValue: "yes",
				Options:      []string{"yes", "no"},
			}) == "yes"

			if deleteDockerfile {
				os.Remove(absDockerfilePath)
				log.Donef("Successfully deleted %s", absDockerfilePath)
			}
		}

		contextPath := "."
		if imageConfig.Build != nil && imageConfig.Build.Context != nil {
			contextPath = *imageConfig.Build.Context
		}

		absContextPath, err := filepath.Abs(contextPath)
		if err != nil {
			continue
		}

		absDockerIgnorePath := filepath.Join(absContextPath, ".dockerignore")
		_, err = os.Stat(absDockerIgnorePath)
		if os.IsNotExist(err) == false {
			deleteDockerIgnore := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:     "\n\nShould " + absDockerIgnorePath + " be removed?",
				DefaultValue: "yes",
				Options:      []string{"yes", "no"},
			}) == "yes"

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
			Question:     "\n\nShould the ClusterRoleBinding '" + clusterRoleBindingName + "' be removed?",
			DefaultValue: "yes",
			Options:      []string{"yes", "no"},
		}) == "yes"

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
		Question:     "\n\nShould the .devspace folder be removed?",
		DefaultValue: "yes",
		Options:      []string{"yes", "no"},
	}) == "yes"

	if deleteDevspaceFolder {
		os.RemoveAll(".devspace")
		log.Done("Successfully deleted .devspace folder")
	}
}
