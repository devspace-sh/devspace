package create

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DevSpaceCloudHostedCluster is the option that is shown during cluster select to select the hosted devspace cloud clusters
const DevSpaceCloudHostedCluster = "DevSpace Cloud Hosted"

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
		Run:  cmd.RunCreateSpace,
	}

	spaceCmd.Flags().BoolVar(&cmd.Active, "active", true, "Use the new Space as active Space for the current project")
	spaceCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")
	spaceCmd.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to create a space in")

	return spaceCmd
}

// RunCreateSpace executes the "devspace create space" command logic
func (cmd *spaceCmd) RunCreateSpace(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	// Check if user has specified a certain provider
	var cloudProvider *string
	if cmd.Provider != "" {
		cloudProvider = &cmd.Provider
	}

	// Get provider
	provider, err := cloud.GetProvider(cloudProvider, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	log.StartWait("Retrieving clusters")
	defer log.StopWait()

	// Get projects
	projects, err := provider.GetProjects()
	if err != nil {
		log.Fatalf("Error retrieving projects: %v", err)
	}

	// Create project if needed
	projectID := 0
	if len(projects) == 0 {
		projectID, err = createProject(provider)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		projectID = projects[0].ProjectID
	}

	var cluster *cloud.Cluster

	if cmd.Cluster == "" {
		cluster, err = getCluster(provider)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cluster, err = provider.GetClusterByName(cmd.Cluster)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.StartWait("Creating space " + args[0])
	defer log.StopWait()

	// Create space
	spaceID, err := provider.CreateSpace(args[0], projectID, cluster)
	if err != nil {
		log.Fatalf("Error creating space: %v", err)
	}

	// Get Space
	space, err := provider.GetSpace(spaceID)
	if err != nil {
		log.Fatalf("Error retrieving space information: %v", err)
	}

	// Get service account
	serviceAccount, err := provider.GetServiceAccount(space)
	if err != nil {
		log.Fatalf("Error retrieving space service account: %v", err)
	}

	// Change kube context
	kubeContext := cloud.GetKubeContextNameFromSpace(space.Name, space.ProviderName)
	err = cloud.UpdateKubeConfig(kubeContext, serviceAccount, true)
	if err != nil {
		log.Fatalf("Error saving kube config: %v", err)
	}

	// Set tiller env
	err = cloud.SetTillerNamespace(serviceAccount)
	if err != nil {
		log.Warnf("Couldn't set tiller namespace environment variable: %v", err)
	}

	// Set space as active space
	if cmd.Active && configExists {
		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		generatedConfig.CloudSpace = &generated.CloudSpaceConfig{
			SpaceID:      space.SpaceID,
			ProviderName: space.ProviderName,
			Name:         space.Name,
			Owner:        space.Owner.Name,
			OwnerID:      space.Owner.OwnerID,
			KubeContext:  kubeContext,
			Created:      space.Created,
			Domain:       space.Domain,
		}

		err = generated.SaveConfig(generatedConfig)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.StopWait()
	log.Infof("Successfully created space %s", space.Name)

	log.Infof("\nYou can now run: \n- `%s` to deploy the app to the cloud\n- `%s` to develop the app in the cloud", ansi.Color("devspace deploy", "white+b"), ansi.Color("devspace dev", "white+b"))
}

func getCluster(p *cloud.Provider) (*cloud.Cluster, error) {
	log.StartWait("Retrieving clusters")
	defer log.StopWait()

	clusters, err := p.GetClusters()
	if err != nil {
		return nil, errors.Wrap(err, "get clusters")
	}
	if len(clusters) == 0 {
		return nil, errors.New("Cannot create space, because no cluster was found")
	}

	log.StopWait()

	// Check if the user has access to a connected cluster
	connectedClusters := make([]*cloud.Cluster, 0, len(clusters))
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

		// Add devspace cloud option
		clusterNames = append(clusterNames, DevSpaceCloudHostedCluster)

		// Choose cluster
		chosenCluster := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:     "Which cluster should the space created in?",
			DefaultValue: clusterNames[0],
			Options:      clusterNames,
		})
		if chosenCluster != DevSpaceCloudHostedCluster {
			for _, cluster := range connectedClusters {
				if cluster.Name == chosenCluster {
					return cluster, nil
				}
			}
		}
	}

	// Select a devspace cluster
	devSpaceClusters := make([]*cloud.Cluster, 0, len(clusters))
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
	chosenCluster := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:     "Which hosted DevSpace cluster should the space created in?",
		DefaultValue: clusterNames[0],
		Options:      clusterNames,
	})
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
