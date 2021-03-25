package add

import (
	"strconv"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type deploymentCmd struct {
	*flags.GlobalFlags

	Manifests string

	Chart        string
	ChartVersion string
	ChartRepo    string

	Image string

	Dockerfile string
	Context    string
}

func newDeploymentCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &deploymentCmd{GlobalFlags: globalFlags}

	addDeploymentCmd := &cobra.Command{
		Use:   "deployment [deployment-name]",
		Short: "Adds a deployment to devspace.yaml",
		Long: ` 
#######################################################
############# devspace add deployment #################
#######################################################
Adds a new deployment to this project's devspace.yaml

Examples:
# Deploy a local dockerfile
devspace add deployment my-deployment --dockerfile=./Dockerfile
devspace add deployment my-deployment --image=myregistry.io/myuser/myrepo --dockerfile=frontend/Dockerfile --context=frontend/Dockerfile
# Deploy an existing docker image
devspace add deployment my-deployment --image=mysql
devspace add deployment my-deployment --image=myregistry.io/myusername/mysql
# Deploy local or remote helm charts
devspace add deployment my-deployment --chart=chart/
devspace add deployment my-deployment --chart=stable/mysql
# Deploy local kubernetes yamls
devspace add deployment my-deployment --manifests=kube/pod.yaml
devspace add deployment my-deployment --manifests=kube/* --namespace=devspace
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAddDeployment(f, cobraCmd, args)
		},
	}

	// Kubectl options
	addDeploymentCmd.Flags().StringVar(&cmd.Manifests, "manifests", "", "The kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. manifests/** or kube/pod.yaml)")

	// Helm chart options
	addDeploymentCmd.Flags().StringVar(&cmd.Chart, "chart", "", "A helm chart to deploy (e.g. ./chart or stable/mysql)")
	addDeploymentCmd.Flags().StringVar(&cmd.ChartVersion, "chart-version", "", "The helm chart version to use")
	addDeploymentCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", "", "The helm chart repository url to use")

	// Component options
	addDeploymentCmd.Flags().StringVar(&cmd.Image, "image", "", "A docker image to deploy (e.g. dscr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)")
	addDeploymentCmd.Flags().StringVar(&cmd.Dockerfile, "dockerfile", "", "A dockerfile")
	addDeploymentCmd.Flags().StringVar(&cmd.Context, "context", "", "")

	return addDeploymentCmd
}

// RunAddDeployment executes the add deployment command logic
func (cmd *deploymentCmd) RunAddDeployment(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	logger := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	logger.Warn("This command is deprecated and will be removed in a future DevSpace version. Please modify the devspace.yaml directly instead")
	deploymentName := args[0]

	// Get base config and check if deployment already exists
	configInterface, err := configLoader.Load(cmd.ToConfigOptions(), logger)
	if err != nil {
		return err
	}

	config := configInterface.Config()
	if config.Deployments != nil {
		for _, deployConfig := range config.Deployments {
			if deployConfig.Name == deploymentName {
				return errors.Errorf("Deployment %s already exists", deploymentName)
			}
		}
	} else {
		config.Deployments = []*latest.DeploymentConfig{}
	}

	configureManager := f.NewConfigureManager(config, logger)

	var newDeployment *latest.DeploymentConfig
	var newImage *latest.ImageConfig

	// figure out what kind of deployment to add
	if cmd.Manifests != "" {
		newDeployment, err = configureManager.NewKubectlDeployment(deploymentName, cmd.Manifests)
	} else if cmd.Chart != "" {
		newDeployment, err = configureManager.NewHelmDeployment(deploymentName, cmd.Chart, cmd.ChartRepo, cmd.ChartVersion)
	} else if cmd.Dockerfile != "" {
		newImage, newDeployment, err = configureManager.NewDockerfileComponentDeployment(deploymentName, cmd.Image, cmd.Dockerfile, cmd.Context)
	} else if cmd.Image != "" {
		newImage, newDeployment, err = configureManager.NewImageComponentDeployment(deploymentName, cmd.Image)
	} else {
		return errors.New("Please specify one of these parameters:\n--image: A docker image to deploy (e.g. dscr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)\n--manifests: The kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. manifests/** or kube/pod.yaml)\n--chart: A helm chart to deploy (e.g. ./chart or stable/mysql)")
	}
	if err != nil {
		return err
	}

	// Add namespace if defined
	if cmd.Namespace != "" {
		newDeployment.Namespace = cmd.Namespace
	}

	// Add image config if necessary
	if newImage != nil {
		imageAlreadyExists := false

		// First check if image already exists in another configuration
		if config.Images != nil {
			for _, imageConfig := range config.Images {
				if imageConfig.Image == newImage.Image {
					imageAlreadyExists = true
					break
				}
			}
		}

		// Only add if it does not already exists
		if imageAlreadyExists == false {
			// Deployment name
			imageName := deploymentName

			// Check if image name exits
			if config.Images != nil {
				for i := 0; true; i++ {
					if _, ok := (config.Images)[imageName]; ok {
						if i == 0 {
							imageName = imageName + "-" + strconv.Itoa(i)
						} else {
							imageName = imageName[:len(imageName)-1] + strconv.Itoa(i)
						}

						continue
					}

					break
				}
			} else {
				config.Images = map[string]*latest.ImageConfig{}
			}

			config.Images[imageName] = newImage
		}
	}

	// Prepend deployment
	if config.Deployments == nil {
		config.Deployments = []*latest.DeploymentConfig{}
	}

	config.Deployments = append([]*latest.DeploymentConfig{newDeployment}, config.Deployments...)

	// Save config
	err = configLoader.Save(config)
	if err != nil {
		return errors.Errorf("Couldn't save config file: %s", err.Error())
	}

	logger.Donef("Successfully added %s as new deployment", args[0])
	return nil
}
