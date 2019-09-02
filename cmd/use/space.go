package use

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type spaceCmd struct {
	Provider string
	SpaceID  string

	GetToken bool
}

func newSpaceCmd() *cobra.Command {
	cmd := &spaceCmd{}

	useSpace := &cobra.Command{
		Use:   "space",
		Short: "Use an existing space for the current configuration",
		Long: `
#######################################################
################ devspace use space ###################
#######################################################
Use an existing space for the current configuration

Example:
devspace use space my-space
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunUseSpace,
	}

	useSpace.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")
	useSpace.Flags().StringVar(&cmd.SpaceID, "space-id", "", "The space id to use")
	useSpace.Flags().BoolVar(&cmd.GetToken, "get-token", false, "Prints the service token to the console")

	return useSpace
}

// RunUseSpace executes the functionality "devspace use space"
func (cmd *spaceCmd) RunUseSpace(cobraCmd *cobra.Command, args []string) {
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

	logger := log.GetInstance()
	if cmd.GetToken == true {
		logger = log.Discard
	}

	// Get cloud provider from config
	provider, err := cloudpkg.GetProvider(cloudProvider, logger)
	if err != nil {
		log.Fatalf("Error getting cloud context: %v", err)
	}
	if provider == nil {
		log.Fatal("No cloud provider specified")
	}

	// List spaces
	if len(args) == 0 && cmd.SpaceID == "" {
		spaces, err := provider.GetSpaces()
		if err != nil {
			log.Fatalf("Error retrieving spaces: %v", err)
		} else if len(spaces) == 0 {
			log.Fatalf("You do not have any Spaces, yet. You can create a space with `%s`", ansi.Color("devspace create space [NAME]", "white+b"))
		}

		names := make([]string, 0, len(spaces))
		for _, space := range spaces {
			names = append(names, space.Name)
		}

		spaceName := survey.Question(&survey.QuestionOptions{
			Question: "Please select the Space that you want to use",
			Options:  names,
		})

		// Set space id
		for _, space := range spaces {
			if space.Name == spaceName {
				cmd.SpaceID = strconv.Itoa(space.SpaceID)
			}
		}
	}

	// Check if we should return a token
	if cmd.GetToken == true {
		spaceID, err := strconv.Atoi(cmd.SpaceID)
		if err != nil {
			log.Fatalf("Error parsing space id: %v", err)
		}

		err = provider.PrintToken(spaceID)
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	log.StartWait("Retrieving Space details")
	var (
		space *cloud.Space
	)

	if len(args) > 0 {
		space, err = provider.GetSpaceByName(args[0])
		if err != nil {
			log.Fatalf("Error retrieving Spaces details: %v", err)
		}
	} else {
		spaceID, err := strconv.Atoi(cmd.SpaceID)
		if err != nil {
			log.Fatalf("Error parsing space id: %v", err)
		}

		space, err = provider.GetSpace(spaceID)
		if err != nil {
			log.Fatalf("Error retrieving Spaces details: %v", err)
		}
	}

	log.StopWait()

	// Get kube context name
	kubeContext := cloud.GetKubeContextNameFromSpace(space.Name, space.ProviderName)

	// Get service account
	serviceAccount, err := provider.GetServiceAccount(space)
	if err != nil {
		log.Fatalf("Error retrieving space service account: %v", err)
	}

	// Change kube context
	err = cloud.UpdateKubeConfig(kubeContext, serviceAccount, space.SpaceID, provider.Name, true)
	if err != nil {
		log.Fatalf("Error saving kube config: %v", err)
	}

	// Set tiller env
	err = cloudpkg.SetTillerNamespace(serviceAccount)
	if err != nil {
		// log.Warnf("Couldn't set tiller namespace environment variable: %v", err)
	}

	if configExists {
		// Signal that we are working on the space if there is any
		err = cloud.ResumeSpace(configutil.GetConfig(), space.ProviderName, space.SpaceID, false, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Donef("Successfully configured DevSpace to use space %s", space.Name)
}
