package create

import (
	"errors"
	"strconv"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type spaceCmd struct {
	context bool
	active  bool
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

	spaceCmd.Flags().BoolVar(&cmd.context, "context", true, "Create/Update kubectl context for space")
	spaceCmd.Flags().BoolVar(&cmd.active, "active", true, "If there is a devspace config, make space the active space")

	return spaceCmd
}

// RunCreateSpace executes the devspace create space command logic
func (cmd *spaceCmd) RunCreateSpace(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}

	// Get provider
	provider, err := cloud.GetCurrentProvider(log.GetInstance())
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
	if cmd.context {
		err = cloud.UpdateKubeConfig(cloud.GetKubeContextNameFromSpace(space), space, true)
		if err != nil {
			log.Fatalf("Error saving kube config: %v", err)
		}
	}

	// Set space as active space
	if cmd.active && configExists {
		// Get generated config
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}

		generatedConfig.Space = space

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
