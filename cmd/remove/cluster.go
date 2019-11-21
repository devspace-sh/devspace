package remove

import (
	"fmt"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type clusterCmd struct {
	Provider string
	AllYes   bool
}

func newClusterCmd() *cobra.Command {
	cmd := &clusterCmd{}

	clusterCmd := &cobra.Command{
		Use:   "cluster",
		Short: "Removes a connected cluster",
		Long: `
#######################################################
############# devspace remove cluster #################
#######################################################
Removes a connected cluster 

Example:
devspace remove cluster my-cluster
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: cmd.RunRemoveCluster,
	}

	clusterCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")
	clusterCmd.Flags().BoolVarP(&cmd.AllYes, "yes", "y", false, "Ignores all questions and deletes the cluster with all services and spaces")

	return clusterCmd
}

// RunRemoveCluster executes the devspace remove cluster functionality
func (cmd *clusterCmd) RunRemoveCluster(cobraCmd *cobra.Command, args []string) error {
	// Get provider
	provider, err := cloudpkg.GetProvider(cmd.Provider, log.GetInstance())
	if err != nil {
		return errors.Wrap(err, "log into provider")
	}

	if cmd.AllYes == false {
		// Verify user is sure to delete the cluster
		deleteCluster, err := survey.Question(&survey.QuestionOptions{
			Question:     fmt.Sprintf("Are you sure you want to delete cluster %s? This action is irreversible", args[0]),
			DefaultValue: "No",
			Options: []string{
				"No",
				"Yes",
			},
		}, log.GetInstance())
		if err != nil {
			return err
		}
		if deleteCluster != "Yes" {
			return nil
		}
	}

	// Get cluster by name
	cluster, err := provider.GetClusterByName(args[0])
	if err != nil {
		return err
	}

	// Delete all spaces?
	var (
		deleteSpaces   = "Yes"
		deleteServices = "Yes"
	)

	if cmd.AllYes == false {
		deleteSpaces, err = survey.Question(&survey.QuestionOptions{
			Question:     "Do you want to delete all cluster spaces?",
			DefaultValue: "No",
			Options: []string{
				"No",
				"Yes",
			},
		}, log.GetInstance())
		if err != nil {
			return err
		}

		// Delete services
		deleteServices, err = survey.Question(&survey.QuestionOptions{
			Question:     "Do you want to delete all cluster services?",
			DefaultValue: "No",
			Options: []string{
				"No",
				"Yes",
			},
		}, log.GetInstance())
		if err != nil {
			return err
		}
	}

	// Delete cluster
	log.StartWait("Deleting cluster " + cluster.Name)
	err = provider.DeleteCluster(cluster, deleteServices == "Yes", deleteSpaces == "Yes")
	if err != nil {
		return err
	}
	log.StopWait()

	delete(provider.ClusterKey, cluster.ClusterID)
	err = provider.Save()
	if err != nil {
		return err
	}

	log.Donef("Successfully deleted cluster %s", args[0])
	return nil
}
