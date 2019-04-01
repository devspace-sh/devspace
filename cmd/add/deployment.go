package add

import (
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/configure"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

type deploymentCmd struct {
	Namespace string
	Manifests string

	Chart        string
	ChartVersion string
	ChartRepo    string

	Image     string
	Component string

	Dockerfile string
	Context    string
}

func newDeploymentCmd() *cobra.Command {
	cmd := &deploymentCmd{}

	addDeploymentCmd := &cobra.Command{
		Use:   "deployment",
		Short: "Add a deployment",
		Long: ` 
#######################################################
############# devspace add deployment #################
#######################################################
Add a new deployment (docker image, components, 
kubernetes manifests or helm chart) to your DevSpace configuration

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
		Run:  cmd.RunAddDeployment,
	}

	addDeploymentCmd.Flags().StringVar(&cmd.Namespace, "namespace", "", "The namespace to use for deploying")

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
func (cmd *deploymentCmd) RunAddDeployment(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if !configExists {
		log.Fatal("Couldn't find a DevSpace configuration. Please run `devspace init`")
	}

	deploymentName := args[0]

	// Get base config and check if deployment already exists
	config := configutil.GetBaseConfig()
	if config.Deployments != nil {
		for _, deployConfig := range *config.Deployments {
			if *deployConfig.Name == deploymentName {
				log.Fatalf("Deployment %s already exists", deploymentName)
			}
		}
	} else {
		config.Deployments = &[]*latest.DeploymentConfig{}
	}

	var newDeployment *latest.DeploymentConfig
	var newImage *latest.ImageConfig

	// figure out what kind of deployment to add
	if cmd.Manifests != "" {
		newDeployment, err = configure.GetKubectlDeployment(deploymentName, cmd.Manifests)
	} else if cmd.Chart != "" {
		newDeployment, err = configure.GetHelmDeployment(deploymentName, cmd.Chart, cmd.ChartRepo, cmd.ChartVersion)
	} else if cmd.Dockerfile != "" {
		newImage, newDeployment, err = configure.GetDockerfileComponentDeployment(deploymentName, cmd.Image, cmd.Dockerfile, cmd.Context)
	} else if cmd.Image != "" {
		newImage, newDeployment, err = configure.GetImageComponentDeployment(deploymentName, cmd.Image)
	} else if cmd.Component != "" {
		newDeployment, err = configure.GetPredefinedComponentDeployment(deploymentName, cmd.Component)
	} else {
		log.Fatal("Please specifiy one of these parameters:\n--image: A docker image to deploy (e.g. dscr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)\n--manifests: The kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. manifests/** or kube/pod.yaml)\n--chart: A helm chart to deploy (e.g. ./chart or stable/mysql)\n--component: A predefined component to use (run `devspace list available-components` to see all available components)")
	}
	if err != nil {
		log.Fatal(err)
	}

	// Add namespace if defined
	if cmd.Namespace != "" {
		newDeployment.Namespace = &cmd.Namespace
	}

	// Prepend deployment
	(*config.Deployments) = append([]*latest.DeploymentConfig{newDeployment}, (*config.Deployments)...)

	// Add image config if necessary
	if newImage != nil {
		imageAlreadyExists := false

		// First check if image already exists in another configuration
		for _, imageConfig := range *config.Images {
			if *imageConfig.Image == *newImage.Image {
				imageAlreadyExists = true
				break
			}
		}

		// Only add if it does not already exist
		if imageAlreadyExists == false {
			// Deployment name
			imageName := deploymentName

			// Check if image name exits
			for i := 0; true; i++ {
				if _, ok := (*config.Images)[imageName]; ok {
					if i == 0 {
						imageName = imageName + "-" + strconv.Itoa(i)
					} else {
						imageName = imageName[:len(imageName)-1] + strconv.Itoa(i)
					}

					continue
				}

				break
			}

			(*config.Images)[imageName] = newImage
		}
	}

	// Save config
	err = configutil.SaveBaseConfig()
	if err != nil {
		log.Fatalf("Couldn't save config file: %s", err.Error())
	}

	log.Donef("Successfully added %s as new deployment", args[0])
}
