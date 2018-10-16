package cmd

import (
	"errors"
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/deploy"
	deployHelm "github.com/covexo/devspace/pkg/devspace/deploy/helm"
	deployKubectl "github.com/covexo/devspace/pkg/devspace/deploy/kubectl"
	helmClient "github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/devspace/registry"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// StatusCmd holds the information needed for the status command
type StatusCmd struct {
	flags   *StatusCmdFlags
	kubectl *kubernetes.Clientset
	workdir string
}

// StatusCmdFlags holds the possible flags for the list command
type StatusCmdFlags struct {
}

func init() {
	cmd := &StatusCmd{
		flags: &StatusCmdFlags{},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Shows the devspace status",
		Long: `
	#######################################################
	################## devspace status ####################
	#######################################################
	Shows the devspace status
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunStatus,
	}

	rootCmd.AddCommand(statusCmd)

	statusSyncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Shows the sync status",
		Long: `
	#######################################################
	################ devspace status sync #################
	#######################################################
	Shows the devspace sync status
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunStatusSync,
	}

	statusCmd.AddCommand(statusSyncCmd)
}

// RunStatus executes the devspace status command logic
func (cmd *StatusCmd) RunStatus(cobraCmd *cobra.Command, args []string) {
	var err error
	var values [][]string
	var headerValues = []string{
		"TYPE",
		"STATUS",
		"NAMESPACE",
		"INFO",
	}
	config := configutil.GetConfig()

	cmd.kubectl, err = kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %s", err.Error())
	}

	if config.Tiller != nil && config.Tiller.Namespace != nil {
		// Check if tiller server is there
		tillerStatus, err := cmd.getTillerStatus()
		if err != nil {
			values = append(values, []string{
				"Tiller",
				"Error",
				"",
				err.Error(),
			})

			log.PrintTable(headerValues, values)
			return
		}

		values = append(values, tillerStatus)
	}

	registryStatus, err := cmd.getRegistryStatus()
	if err != nil {
		values = append(values, []string{
			"Internal Registry",
			"Not Deployed",
			"",
			"",
			err.Error(),
		})
	} else if registryStatus != nil {
		values = append(values, registryStatus)
	}

	if config.DevSpace != nil && config.DevSpace.Deployments != nil {
		for _, deployConfig := range *config.DevSpace.Deployments {
			var deployClient deploy.Interface

			// Delete kubectl engine
			if deployConfig.Kubectl != nil {
				deployClient, err = deployKubectl.New(cmd.kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create kubectl deploy config for %s: %v", *deployConfig.Name, err)
					continue
				}
			} else {
				deployClient, err = deployHelm.New(cmd.kubectl, deployConfig, log.GetInstance())
				if err != nil {
					log.Warnf("Unable to create helm deploy config for %s: %v", *deployConfig.Name, err)
					continue
				}
			}

			addValues, err := deployClient.Status()
			if err != nil {
				log.Warnf("Error retrieving status for deployment %s: %v", *deployConfig.Name, err)
			}

			values = append(values, addValues...)
		}
	}

	log.PrintTable(headerValues, values)
}

func (cmd *StatusCmd) getTillerStatus() ([]string, error) {
	config := configutil.GetConfig()
	tillerNamespace := *config.Tiller.Namespace

	tillerPod, err := kubectl.GetPodsFromDeployment(cmd.kubectl, helmClient.TillerDeploymentName, tillerNamespace)
	if err != nil {
		return nil, err
	}
	if len(tillerPod.Items) == 0 {
		return nil, errors.New("No tiller pod found")
	}

	for _, pod := range tillerPod.Items {
		if kubectl.GetPodStatus(&pod) == "Running" {
			return []string{
				"Tiller",
				"Running",
				pod.GetNamespace(),
				"",
			}, nil
		}
	}

	return nil, errors.New("No running tiller pod found")
}

func (cmd *StatusCmd) getRegistryStatus() ([]string, error) {
	config := configutil.GetConfig()
	registryConfig := config.InternalRegistry
	if registryConfig == nil {
		return nil, nil
	}

	helm, err := helmClient.NewClient(cmd.kubectl, log.GetInstance(), false)
	if err != nil {
		return nil, err
	}

	releases, err := helm.Client.ListReleases()
	if err != nil {
		return nil, err
	}

	if len(releases.Releases) == 0 {
		return nil, errors.New("No release found")
	}

	for _, release := range releases.Releases {
		if release.GetName() == registry.InternalRegistryName {
			if release.Info.Status.Code.String() != "DEPLOYED" {
				return nil, fmt.Errorf("Registry helm release has bad status: %s", release.Info.Status.Code.String())
			}

			registryPods, err := kubectl.GetPodsFromDeployment(cmd.kubectl, registry.InternalRegistryDeploymentName, *registryConfig.Namespace)

			if err != nil {
				return nil, err
			}

			if len(registryPods.Items) == 0 {
				return nil, errors.New("No registry pods found")
			}

			for _, pod := range registryPods.Items {
				if kubectl.GetPodStatus(&pod) == "Running" {
					return []string{
						"Internal Registry",
						"Running",
						pod.GetName(),
						pod.GetNamespace(),
						"",
						//fmt.Sprintf("Created: %s", pod.GetCreationTimestamp().String()),
					}, nil
				}
			}

			return nil, errors.New("No running registry pod found")
		}
	}

	return nil, fmt.Errorf("Registry helm release %s not found", registry.InternalRegistryName)
}
