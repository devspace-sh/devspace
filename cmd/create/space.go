package create

import (
	"errors"
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type spaceCmd struct {
	active   bool
	provider string
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

	spaceCmd.Flags().BoolVar(&cmd.active, "active", true, "Use the new Space as active Space for the current project")
	spaceCmd.Flags().StringVar(&cmd.provider, "provider", "", "The cloud provider to use")

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
	if cmd.provider != "" {
		cloudProvider = &cmd.provider
	}

	// Get provider
	provider, err := cloud.GetProvider(cloudProvider, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	log.StartWait("Creating space " + args[0])
	defer log.StopWait()

	// Get projects
	projects, err := provider.GetProjects()
	if err != nil {
		log.Fatal(err)
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

	log.StartWait("Creating space " + args[0])
	defer log.StopWait()

	// Create space
	spaceID, err := provider.CreateSpace(args[0], projectID, nil)
	if err != nil {
		log.Fatalf("Error creating space: %v", err)
	}

	// Get Space
	space, err := provider.GetSpace(spaceID)
	if err != nil {
		log.Fatalf("Error retrieving space information: %v", err)
	}

	// Change kube context
	kubeContext := cloud.GetKubeContextNameFromSpace(space.Name, space.ProviderName)
	err = cloud.UpdateKubeConfig(kubeContext, space, true)
	if err != nil {
		log.Fatalf("Error saving kube config: %v", err)
	}

	// Set tiller env
	err = cloud.SetTillerNamespace(space)
	if err != nil {
		// log.Warnf("Couldn't set tiller namespace environment variable: %v", err)
	}

	// Set space as active space
	if cmd.active && configExists {
		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		generatedConfig.CloudSpace = &generated.CloudSpaceConfig{
			SpaceID:      space.SpaceID,
			ProviderName: space.ProviderName,
			Name:         space.Name,
			Namespace:    space.Namespace,
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

func createProject(p *cloud.Provider) (int, error) {
	clusters, err := p.GetClusters()
	if err != nil {
		return 0, err
	}
	if len(clusters) == 0 {
		return 0, errors.New("Cannot create project, because no public cluster was found")
	}

	clusterID := clusters[0].ClusterID
	if len(clusters) > 1 {
		clusterNames := map[string]*cloud.Cluster{}
		for idx, cluster := range clusters {
			if cluster.Name == nil {
				clusterNames["Cluster-"+strconv.Itoa(idx)] = cluster
			} else {
				clusterNames[*cluster.Name] = cluster
			}
		}

		clustersArr := []string{}
		for name := range clusterNames {
			clustersArr = append(clustersArr, name)
		}

		log.StopWait()
		chosenCluster := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:     "Which cluster do you want to use?",
			DefaultValue: clustersArr[0],
			Options:      clustersArr,
		})

		clusterID = clusterNames[chosenCluster].ClusterID
	}

	projectID, err := p.CreateProject("default", clusterID)
	if err != nil {
		return 0, err
	}

	return projectID, err
}
