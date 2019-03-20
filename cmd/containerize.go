package cmd

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

// ContainerizeCmd holds the containerize cmd flags
type ContainerizeCmd struct {
	Path string
}

// NewContainerizeCmd creates a new ui command
func NewContainerizeCmd() *cobra.Command {
	cmd := &ContainerizeCmd{}

	containerizeCmd := &cobra.Command{
		Use:   "containerize",
		Short: "Creates a Dockerfile in the project",
		Long: `
#######################################################
################ devspace containerize ################
#######################################################
Creates a dockerfile in the project based on the
detected programming language.

Examples:
devspace containerize
devspace containerize --path=./Dockerfile.development
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunContainerize,
	}

	containerizeCmd.Flags().StringVar(&cmd.Path, "path", "./Dockerfile", "The path to use")

	return containerizeCmd
}

// RunContainerize executes the functionality "devspace containerize"
func (cmd *ContainerizeCmd) RunContainerize(cobraCmd *cobra.Command, args []string) {
	// Print DevSpace logo
	log.PrintLogo()

	// Containerize application if necessary
	err := generator.ContainerizeApplication(cmd.Path, ".", "")
	if err != nil {
		log.Fatalf("Error containerizing application: %v", err)
	}

	log.Infof("Successfully containerized project. Run: \n- `%s` to initialize DevSpace in the project\n- `%s` to verify that the Dockerfile is working in this project", ansi.Color("devspace init", "white+b"), ansi.Color("docker build .", "white+b"))
}
