package add

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type imageCmd struct {
	Name           string
	Tag            string
	ContextPath    string
	DockerfilePath string
	BuildEngine    string
}

func newImageCmd() *cobra.Command {
	cmd := &imageCmd{}

	addImageCmd := &cobra.Command{
		Use:   "image",
		Short: "Add an image",
		Long: ` 
#######################################################
############# devspace add image ######################
#######################################################
Add a new image to your DevSpace configuration

Examples:
devspace add image my-image --image=dockeruser/devspaceimage2
devspace add image my-image --image=dockeruser/devspaceimage2 --tag=alpine
devspace add image my-image --image=dockeruser/devspaceimage2 --context=./context
devspace add image my-image --image=dockeruser/devspaceimage2 --dockerfile=./Dockerfile
devspace add image my-image --image=dockeruser/devspaceimage2 --buildengine=docker
devspace add image my-image --image=dockeruser/devspaceimage2 --buildengine=kaniko
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		Run:  cmd.RunAddImage,
	}

	addImageCmd.Flags().StringVar(&cmd.Name, "image", "", "The image name of the image (e.g. myusername/devspace)")
	addImageCmd.Flags().StringVar(&cmd.Tag, "tag", "", "The tag of the image")
	addImageCmd.Flags().StringVar(&cmd.ContextPath, "context", "", "The path of the images' context")
	addImageCmd.Flags().StringVar(&cmd.DockerfilePath, "dockerfile", "", "The path of the images' dockerfile")
	addImageCmd.Flags().StringVar(&cmd.BuildEngine, "buildengine", "", "Specify which engine should build the file. Should match this regex: docker|kaniko")

	addImageCmd.MarkFlagRequired("image")
	return addImageCmd
}

// RunAddImage executes the add image command logic
func (cmd *imageCmd) RunAddImage(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	config := configutil.GetBaseConfig("")

	err = configure.AddImage(config, args[0], cmd.Name, cmd.Tag, cmd.ContextPath, cmd.DockerfilePath, cmd.BuildEngine)
	if err != nil {
		log.Fatal(err)
	}

	log.Donef("Successfully added image %s", args[0])
}
