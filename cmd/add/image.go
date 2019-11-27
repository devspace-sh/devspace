package add

import (
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type imageCmd struct {
	*flags.GlobalFlags

	Name           string
	Tag            string
	ContextPath    string
	DockerfilePath string
	BuildTool      string
}

func newImageCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &imageCmd{GlobalFlags: globalFlags}

	addImageCmd := &cobra.Command{
		Use:   "image",
		Short: "Add an image",
		Long: ` 
#######################################################
############# devspace add image ######################
#######################################################
Adds a new image to this project's devspace.yaml

Examples:
devspace add image my-image --image=dockeruser/devspaceimage2
devspace add image my-image --image=dockeruser/devspaceimage2 --tag=alpine
devspace add image my-image --image=dockeruser/devspaceimage2 --context=./context
devspace add image my-image --image=dockeruser/devspaceimage2 --dockerfile=./Dockerfile
devspace add image my-image --image=dockeruser/devspaceimage2 --buildtool=docker
devspace add image my-image --image=dockeruser/devspaceimage2 --buildtool=kaniko
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: cmd.RunAddImage,
	}

	addImageCmd.Flags().StringVar(&cmd.Name, "image", "", "The image name of the image (e.g. myusername/devspace)")
	addImageCmd.Flags().StringVar(&cmd.Tag, "tag", "", "The tag of the image")
	addImageCmd.Flags().StringVar(&cmd.ContextPath, "context", "", "The path of the images' context")
	addImageCmd.Flags().StringVar(&cmd.DockerfilePath, "dockerfile", "", "The path of the images' dockerfile")
	addImageCmd.Flags().StringVar(&cmd.BuildTool, "buildtool", "", "Specify which engine should build the file. Should match this regex: docker|kaniko")

	addImageCmd.MarkFlagRequired("image")
	return addImageCmd
}

// RunAddImage executes the add image command logic
func (cmd *imageCmd) RunAddImage(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log.GetInstance())
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

	err = configure.AddImage(config, args[0], cmd.Name, cmd.Tag, cmd.ContextPath, cmd.DockerfilePath, cmd.BuildTool, log.GetInstance())
	if err != nil {
		return err
	}

	err = configLoader.Save(config)
	if err != nil {
		return err
	}

	log.GetInstance().Donef("Successfully added image %s", args[0])
	return nil
}
