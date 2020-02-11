package remove

import (
	"errors"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/spf13/cobra"
)

type imageCmd struct {
	*flags.GlobalFlags

	RemoveAll bool
}

func newImageCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &imageCmd{GlobalFlags: globalFlags}

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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunRemoveImage(f, cobraCmd, args)
		}}

	imageCmd.Flags().BoolVar(&cmd.RemoveAll, "all", false, "Remove all images")

	return imageCmd
}

// RunRemoveImage executes the remove image command logic
func (cmd *imageCmd) RunRemoveImage(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	log := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ToConfigOptions(), log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	config, err := configLoader.LoadWithoutProfile()
	if err != nil {
		return err
	}

	configureManager := f.NewConfigureManager(config, log)
	err = configureManager.RemoveImage(cmd.RemoveAll, args)
	if err != nil {
		return err
	}

	err = configLoader.Save(config)
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
