package remove

import (
	"errors"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type imageCmd struct {
	RemoveAll bool
}

func newImageCmd() *cobra.Command {
	cmd := &imageCmd{}

	imageCmd := &cobra.Command{
		Use:   "image",
		Short: "Removes one or all images from the devspace",
		Long: `
#######################################################
############ devspace remove image ####################
#######################################################
Removes one or all images from a devspace:
devspace remove image default
devspace remove image --all
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		RunE: cmd.RunRemoveImage,
	}

	imageCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all images")

	return imageCmd
}

// RunRemoveImage executes the remove image command logic
func (cmd *imageCmd) RunRemoveImage(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New("Couldn't find any devspace configuration. Please run `devspace init`")
	}

	config, err := configutil.GetBaseConfig("")
	if err != nil {
		return err
	}

	err = configure.RemoveImage(config, cmd.RemoveAll, args)
	if err != nil {
		return err
	}

	if cmd.RemoveAll {
		log.Done("Successfully removed all images")
	} else {
		log.Donef("Successfully removed image %s", args[0])
	}

	return nil
}
