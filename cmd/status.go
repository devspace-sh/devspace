package cmd

import (
	"errors"
	"fmt"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	helmClient "github.com/covexo/devspace/pkg/devspace/deploy/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/daviddengcn/go-colortext"
	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// StatusCmd holds the information needed for the status command
type StatusCmd struct {
	flags   *StatusCmdFlags
	helm    *helmClient.HelmClientWrapper
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
		"POD",
		"NAMESPACE",
		"INFO",
	}

	cmd.kubectl, err = kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %s", err.Error())
	}

	// Check if tiller server is there
	tillerStatus, err := cmd.getTillerStatus()
	if err != nil {
		values = append(values, []string{
			"Tiller",
			"Error",
			"",
			"",
			err.Error(),
		})

		log.PrintTable(headerValues, values)
		return
	}

	values = append(values, tillerStatus)
	cmd.helm, err = helmClient.NewClient(cmd.kubectl, false)
	if err != nil {
		log.Fatalf("Error initializing helm client: %s", err.Error())
	}

	registryStatus, err := cmd.getRegistryStatus()

	if err != nil {
		values = append(values, []string{
			"Docker Registry",
			"Not Deployed",
			"",
			"",
			err.Error(),
		})
	} else if registryStatus != nil {
		values = append(values, registryStatus)
	}

	devspaceStatus, err := cmd.getDevspaceStatus()

	if err != nil {
		values = append(values, []string{
			"Devspace",
			"Error",
			"",
			"",
			err.Error(),
		})

		log.PrintTable(headerValues, values)

		// Print Describes of failed devspace pods
		if devspaceStatus != nil {
			log.Info("Below details of the not running devspace pods are shown")

			for k, v := range devspaceStatus {
				if k > 0 {
					log.WriteColored("--------------------------------------------------------\n", ct.Green)
				}

				log.Write("\n" + v + "\n\n")
			}
		}
	} else {
		values = append(values, devspaceStatus)

		log.PrintTable(headerValues, values)
	}
}

func (cmd *StatusCmd) getRegistryStatus() ([]string, error) {
	config := configutil.GetConfig()
	registry := config.Services.InternalRegistry
	if registry == nil {
		return nil, nil
	}

	releases, err := cmd.helm.Client.ListReleases()

	if err != nil {
		return nil, err
	}

	if len(releases.Releases) == 0 {
		return nil, errors.New("No release found")
	}

	for _, release := range releases.Releases {
		if release.GetName() == *registry.Release.Name {
			if release.Info.Status.Code.String() != "DEPLOYED" {
				return nil, fmt.Errorf("Registry helm release has bad status: %s", release.Info.Status.Code.String())
			}

			registryPods, err := kubectl.GetPodsFromDeployment(cmd.kubectl, *registry.Release.Name+"-docker-registry", *registry.Release.Namespace)

			if err != nil {
				return nil, err
			}

			if len(registryPods.Items) == 0 {
				return nil, errors.New("No registry pods found")
			}

			for _, pod := range registryPods.Items {
				if kubectl.GetPodStatus(&pod) == "Running" {
					return []string{
						"Docker Registry",
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

	return nil, fmt.Errorf("Registry helm release %s not found", *registry.Release.Name)
}

func (cmd *StatusCmd) getTillerStatus() ([]string, error) {
	config := configutil.GetConfig()
	tillerPod, err := kubectl.GetPodsFromDeployment(cmd.kubectl, helmClient.TillerDeploymentName, *config.Services.Tiller.Release.Namespace)

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
				pod.GetName(),
				pod.GetNamespace(),
				"",
				//fmt.Sprintf("Created: %s", pod.GetCreationTimestamp().String()),
			}, nil
		}
	}

	return nil, errors.New("No running tiller pod found")
}

func (cmd *StatusCmd) getDevspaceStatus() ([]string, error) {
	config := configutil.GetConfig()
	releases, err := cmd.helm.Client.ListReleases()

	if err != nil {
		return nil, err
	}

	if releases == nil || len(releases.Releases) == 0 {
		return nil, errors.New("No release found")
	}

	for _, release := range releases.Releases {
		if release.GetName() == *config.DevSpace.Release.Name {
			if release.Info.Status.Code.String() != "DEPLOYED" {
				return nil, fmt.Errorf("Devspace helm release %s has bad status: %s", *config.DevSpace.Release.Name, release.Info.Status.Code.String())
			}

			pods, err := cmd.kubectl.Core().Pods(*config.DevSpace.Release.Namespace).List(metav1.ListOptions{
				LabelSelector: "release=" + *config.DevSpace.Release.Name,
			})

			if err != nil {
				return nil, err
			}

			if len(pods.Items) == 0 {
				return nil, errors.New("No devspace pod found")
			}

			for _, pod := range pods.Items {
				// Print Describe on devspace error
				if kubectl.GetPodStatus(&pod) == "Running" {
					return []string{
						"Devspace",
						"Running",
						pod.GetName(),
						pod.GetNamespace(),
						"",
						// fmt.Sprintf("Created: %s", pod.GetCreationTimestamp().String()),
					}, nil
				}
			}

			describe := make([]string, 0, len(pods.Items))

			for _, pod := range pods.Items {
				describeString, err := kubectl.DescribePod(pod.GetNamespace(), pod.GetName())

				if err == nil {
					describe = append(describe, describeString)
				}
			}

			return describe, errors.New("No running devspace pod found")
		}
	}

	return nil, fmt.Errorf("Devspace helm release %s not found", *config.DevSpace.Release.Name)
}

func getRunningDevSpacePod(helm *helmClient.HelmClientWrapper, client *kubernetes.Clientset) (*k8sv1.Pod, error) {
	config := configutil.GetConfig()
	releases, err := helm.Client.ListReleases()

	if err != nil {
		return nil, err
	}

	if releases == nil || len(releases.Releases) == 0 {
		return nil, errors.New("No release found")
	}

	for _, release := range releases.Releases {
		if release.GetName() == *config.DevSpace.Release.Name {
			if release.Info.Status.Code.String() != "DEPLOYED" {
				return nil, fmt.Errorf("Devspace helm release %s has bad status: %s", *config.DevSpace.Release.Name, release.Info.Status.Code.String())
			}

			pods, err := client.Core().Pods(*config.DevSpace.Release.Namespace).List(metav1.ListOptions{
				LabelSelector: "release=" + *config.DevSpace.Release.Name,
			})

			if err != nil {
				return nil, err
			}

			if len(pods.Items) == 0 {
				return nil, errors.New("No devspace pod found")
			}

			for _, pod := range pods.Items {
				// Print Describe on devspace error
				if kubectl.GetPodStatus(&pod) == "Running" {
					return &pod, nil
				}
			}

			return nil, errors.New("No running devspace pod found")
		}
	}

	return nil, fmt.Errorf("Devspace helm release %s not found", *config.DevSpace.Release.Name)
}
