package create

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DevSpaceCloudHostedCluster is the option that is shown during cluster select to select the hosted devspace cloud clusters
const DevSpaceCloudHostedCluster = "Clusters managed by DevSpace"

type spaceCmd struct {
	Active   bool
	Provider string
	Cluster  string
}

func newSpaceCmd() *cobra.Command {
	cmd := &spaceCmd{}

	spaceCmd := &cobra.Command{
		Use:   "space",
		Short: "Create a new cloud space",
		Long: `
#######################################################
############### devspace create space #################
#######################################################
Creates a new space

Example:
devspace create space myspace
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: cmd.RunCreateSpace,
	}

	spaceCmd.Flags().BoolVar(&cmd.Active, "active", true, "Use the new Space as active Space for the current project")
	spaceCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")
	spaceCmd.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to create a space in")

	return spaceCmd
}

// RunCreateSpace executes the "devspace create space" command logic
func (cmd *spaceCmd) RunCreateSpace(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot(log.GetInstance())
	if err != nil {
		return err
	}

	// Get provider
	provider, err := cloud.GetProvider(cmd.Provider, log.GetInstance())
	if err != nil {
		return err
	}

	log.StartWait("Retrieving clusters")
	defer log.StopWait()

	// Get projects
	projects, err := provider.GetProjects()
	if err != nil {
		return errors.Wrap(err, "get projects")
	}

	// Create project if needed
	projectID := 0
	if len(projects) == 0 {
		projectID, err = createProject(provider)
		if err != nil {
			return err
		}
	} else {
		projectID = projects[0].ProjectID
	}

	var cluster *latest.Cluster
	if cmd.Cluster == "" {
		cluster, err = getCluster(provider)
		if err != nil {
			return err
		}
	} else {
		cluster, err = provider.GetClusterByName(cmd.Cluster)
		if err != nil {
			return err
		}
	}

	log.StartWait("Creating space " + args[0])
	defer log.StopWait()

	// Create space
	spaceID, err := provider.CreateSpace(args[0], projectID, cluster)
	if err != nil {
		return errors.Wrap(err, "create space")
	}

	// Get Space
	space, err := provider.GetSpace(spaceID)
	if err != nil {
		return errors.Wrap(err, "get space")
	}

	// Get service account
	serviceAccount, err := provider.GetServiceAccount(space)
	if err != nil {
		return errors.Wrap(err, "get serviceaccount")
	}

	// Change kube context
	kubeContext := cloud.GetKubeContextNameFromSpace(space.Name, space.ProviderName)
	err = cloud.UpdateKubeConfig(kubeContext, serviceAccount, spaceID, provider.Name, true)
	if err != nil {
		return errors.Wrap(err, "update kube context")
	}

	// Cache space
	err = provider.CacheSpace(space, serviceAccount)
	if err != nil {
		return err
	}

	log.StopWait()
	log.Infof("Successfully created space %s", space.Name)
	log.Infof("Your kubectl context has been updated automatically.")

	if configExists {
		log.Infof("\r         \nYou can now run: \n- `%s` to deploy the app to the cloud\n- `%s` to develop the app in the cloud\n", ansi.Color("devspace deploy", "white+b"), ansi.Color("devspace dev", "white+b"))
	}

	return nil
}

func getCluster(p *cloud.Provider) (*latest.Cluster, error) {

	clusters, err := p.GetClusters()
	if err != nil {
		return nil, errors.Wrap(err, "get clusters")
	}
	if len(clusters) == 0 {
		return nil, errors.New("Cannot create space, because no cluster was found")
	}

	log.StopWait()

	// Check if the user has access to a connected cluster
	connectedClusters := make([]*latest.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		if cluster.Owner != nil {
			connectedClusters = append(connectedClusters, cluster)
		}
	}

	// Check if user has connected clusters
	if len(connectedClusters) > 0 {
		clusterNames := []string{}
		for _, cluster := range connectedClusters {
			clusterNames = append(clusterNames, cluster.Name)
		}

		// Check if there are non connected clusters
		for _, cluster := range clusters {
			if cluster.Owner == nil {
				// Add devspace cloud option
				clusterNames = append(clusterNames, DevSpaceCloudHostedCluster)
				break
			}
		}
		if len(clusterNames) == 1 {
			return connectedClusters[0], nil
		}

		// Choose cluster
		chosenCluster, err := survey.Question(&survey.QuestionOptions{
			Question:     "Which cluster should the space created in?",
			DefaultValue: clusterNames[0],
			Options:      clusterNames,
		}, log.GetInstance())
		if err != nil {
			return nil, err
		}

		if chosenCluster != DevSpaceCloudHostedCluster {
			for _, cluster := range connectedClusters {
				if cluster.Name == chosenCluster {
					return cluster, nil
				}
			}
		}
	}

	// Select a devspace cluster
	devSpaceClusters := make([]*latest.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		if cluster.Owner == nil {
			devSpaceClusters = append(devSpaceClusters, cluster)
		}
	}

	if len(devSpaceClusters) == 1 {
		return devSpaceClusters[0], nil
	}

	clusterNames := []string{}
	for _, cluster := range devSpaceClusters {
		clusterNames = append(clusterNames, cluster.Name)
	}

	// Choose cluster
	chosenCluster, err := survey.Question(&survey.QuestionOptions{
		Question:     "Which hosted DevSpace cluster should the space created in?",
		DefaultValue: clusterNames[0],
		Options:      clusterNames,
	}, log.GetInstance())
	if err != nil {
		return nil, err
	}

	for _, cluster := range devSpaceClusters {
		if cluster.Name == chosenCluster {
			return cluster, nil
		}
	}

	return nil, errors.New("No cluster selected")
}

func createProject(p *cloud.Provider) (int, error) {
	return p.CreateProject("default")
}
