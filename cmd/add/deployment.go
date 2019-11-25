package add

import (
	"strconv"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type deploymentCmd struct {
	*flags.GlobalFlags

	Manifests string

	Chart        string
	ChartVersion string
	ChartRepo    string

	Image     string
	Component string

	Dockerfile string
	Context    string
}

func newDeploymentCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
# Deploy a predefined component 
devspace add deployment my-deployment --component=mysql
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
		RunE: cmd.RunAddDeployment,
	}

	// Kubectl options
	addDeploymentCmd.Flags().StringVar(&cmd.Manifests, "manifests", "", "The kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. manifests/** or kube/pod.yaml)")

	// Helm chart options
	addDeploymentCmd.Flags().StringVar(&cmd.Chart, "chart", "", "A helm chart to deploy (e.g. ./chart or stable/mysql)")
	addDeploymentCmd.Flags().StringVar(&cmd.ChartVersion, "chart-version", "", "The helm chart version to use")
	addDeploymentCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", "", "The helm chart repository url to use")

	// Component options
	addDeploymentCmd.Flags().StringVar(&cmd.Image, "image", "", "A docker image to deploy (e.g. dscr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)")
	addDeploymentCmd.Flags().StringVar(&cmd.Component, "component", "", "A predefined component to use (run `devspace list available-components` to see all available components)")
	addDeploymentCmd.Flags().StringVar(&cmd.Dockerfile, "dockerfile", "", "A dockerfile")
	addDeploymentCmd.Flags().StringVar(&cmd.Context, "context", "", "")

	return addDeploymentCmd
}

// RunAddDeployment executes the add deployment command logic
func (cmd *deploymentCmd) RunAddDeployment(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), log.GetInstance())
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	deploymentName := args[0]

	// Get base config and check if deployment already exists
	config, err := configLoader.LoadWithoutProfile()
	if err != nil {
		return err
	}
	if config.Deployments != nil {
		for _, deployConfig := range config.Deployments {
			if deployConfig.Name == deploymentName {
				return errors.Errorf("Deployment %s already exists", deploymentName)
			}
		}
	} else {
		config.Deployments = []*latest.DeploymentConfig{}
	}

	var newDeployment *latest.DeploymentConfig
	var newImage *latest.ImageConfig

	// figure out what kind of deployment to add
	if cmd.Manifests != "" {
		newDeployment, err = configure.GetKubectlDeployment(deploymentName, cmd.Manifests)
	} else if cmd.Chart != "" {
		newDeployment, err = configure.GetHelmDeployment(deploymentName, cmd.Chart, cmd.ChartRepo, cmd.ChartVersion)
	} else if cmd.Dockerfile != "" {
		generatedConfig, err := configLoader.Generated()
		if err != nil {
			return err
		}

		newImage, newDeployment, err = configure.GetDockerfileComponentDeployment(config, generatedConfig, deploymentName, cmd.Image, cmd.Dockerfile, cmd.Context, log.GetInstance())
	} else if cmd.Image != "" {
		newImage, newDeployment, err = configure.GetImageComponentDeployment(deploymentName, cmd.Image, log.GetInstance())
	} else if cmd.Component != "" {
		newDeployment, err = configure.GetPredefinedComponentDeployment(deploymentName, cmd.Component, log.GetInstance())
	} else {
		return errors.New("Please specifiy one of these parameters:\n--image: A docker image to deploy (e.g. dscr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)\n--manifests: The kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. manifests/** or kube/pod.yaml)\n--chart: A helm chart to deploy (e.g. ./chart or stable/mysql)\n--component: A predefined component to use (run `devspace list available-components` to see all available components)")
	}
	if err != nil {
		return err
	}

	// Add namespace if defined
	if cmd.Namespace != "" {
		newDeployment.Namespace = cmd.Namespace
	}

	// Restore vars in config
	clonedConfig, err := configLoader.RestoreVars(config)
	if err != nil {
		return errors.Errorf("Error restoring vars: %v", err)
	}

	// Add image config if necessary
	if newImage != nil {
		imageAlreadyExists := false

		// First check if image already exists in another configuration
		if clonedConfig.Images != nil {
			for _, imageConfig := range clonedConfig.Images {
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
			if clonedConfig.Images != nil {
				for i := 0; true; i++ {
					if _, ok := (clonedConfig.Images)[imageName]; ok {
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
				clonedConfig.Images = map[string]*latest.ImageConfig{}
			}

			(clonedConfig.Images)[imageName] = newImage
		}
	}

	// Prepend deployment
	if clonedConfig.Deployments == nil {
		clonedConfig.Deployments = []*latest.DeploymentConfig{}
	}

	clonedConfig.Deployments = append([]*latest.DeploymentConfig{newDeployment}, clonedConfig.Deployments...)

	// Save config
	err = configLoader.Save(clonedConfig)
	if err != nil {
		return errors.Errorf("Couldn't save config file: %s", err.Error())
	}

	log.GetInstance().Donef("Successfully added %s as new deployment", args[0])
	return nil
}
