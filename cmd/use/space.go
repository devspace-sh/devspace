package use

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/resume"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
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
		RunE: cmd.RunUseSpace,
	}

	useSpace.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")
	useSpace.Flags().StringVar(&cmd.SpaceID, "space-id", "", "The space id to use")
	useSpace.Flags().BoolVar(&cmd.GetToken, "get-token", false, "Prints the service token to the console")

	return useSpace
}

// RunUseSpace executes the functionality "devspace use space"
func (cmd *spaceCmd) RunUseSpace(cobraCmd *cobra.Command, args []string) error {
	log := logpkg.GetInstance()

	// Set config root
	configExists, err := loader.NewConfigLoader(nil, log).SetDevSpaceRoot()
	if err != nil {
		return err
	}

	logger := log
	if cmd.GetToken == true {
		logger = logpkg.Discard
	}

	// Get cloud provider from config
	provider, err := cloud.GetProvider(cmd.Provider, logger)
	if err != nil {
		return errors.Wrap(err, "get provider")
	}
	if provider == nil {
		return errors.New("No cloud provider specified")
	}

	// List spaces
	if len(args) == 0 && cmd.SpaceID == "" {
		spaces, err := provider.Client().GetSpaces()
		if err != nil {
			return errors.Errorf("Error retrieving spaces: %v", err)
		} else if len(spaces) == 0 {
			return errors.Errorf("You do not have any Spaces, yet. You can create a space with `%s`", ansi.Color("devspace create space [NAME]", "white+b"))
		}

		names := make([]string, 0, len(spaces))
		for _, space := range spaces {
			names = append(names, space.Name)
		}

		spaceName, err := log.Question(&survey.QuestionOptions{
			Question: "Please select the Space that you want to use",
			Options:  names,
		})
		if err != nil {
			return err
		}

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
			return errors.Wrap(err, "parse space id")
		}

		return provider.PrintToken(spaceID)
	}

	log.StartWait("Retrieving Space details")
	var (
		space *latest.Space
	)

	if len(args) > 0 {
		space, err = provider.Client().GetSpaceByName(args[0])
		if err != nil {
			return errors.Errorf("%s: %v", message.SpaceQueryError, err)
		}
	} else {
		spaceID, err := strconv.Atoi(cmd.SpaceID)
		if err != nil {
			return errors.Errorf("Error parsing space id: %v", err)
		}

		space, err = provider.Client().GetSpace(spaceID)
		if err != nil {
			return errors.Errorf("%s: %v", message.SpaceQueryError, err)
		}
	}

	log.StopWait()

	// Get kube context name
	kubeContext := cloud.GetKubeContextNameFromSpace(space.Name, space.ProviderName)

	// Get cluster key
	key, err := provider.GetClusterKey(space.Cluster)
	if err != nil {
		return errors.Wrap(err, "get cluster key")
	}

	// Get service account
	serviceAccount, err := provider.Client().GetServiceAccount(space, key)
	if err != nil {
		return errors.Errorf("Error retrieving space service account: %v", err)
	}

	// Change kube context
	err = provider.UpdateKubeConfig(kubeContext, serviceAccount, space.SpaceID, true)
	if err != nil {
		return errors.Errorf("Error saving kube config: %v", err)
	}

	// Cache space
	err = provider.CacheSpace(space, serviceAccount)
	if err != nil {
		return err
	}

	client, err := kubectl.NewClientFromContext(kubeContext, "", false)
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = resume.NewSpaceResumer(client, log).ResumeSpace(false)
	if err != nil {
		return err
	}

	log.Donef("Successfully configured DevSpace to use space %s", space.Name)
	if configExists {
		log.Infof("\r         \nRun:\n- `%s` to develop application\n- `%s` to deploy application\n", ansi.Color("devspace dev", "white+b"), ansi.Color("devspace deploy", "white+b"))
	}

	return nil
}
