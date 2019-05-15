package docker

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/otiai10/copy"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
)

//@Moretest
//Coverage is 46% and that's not enough

func TestDockerBuild(t *testing.T) {
	t.Skip("For some reason there's a problem with the coverage in this package")

	// 1. Write test dockerfile and context to a temp folder
	dir, err := ioutil.TempDir("", "testDocker")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	copy.Copy("./../../../../examples/minikube", dir)

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// 4. Cleanup temp folder
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	deployConfig := &latest.DeploymentConfig{
		Name: ptr.String("test-deployment"),
		Component: &latest.ComponentConfig{
			Containers: &[]*latest.ContainerConfig{
				{
					Image: ptr.String("nginx"),
				},
			},
			Service: &latest.ServiceConfig{
				Ports: &[]*latest.ServicePortConfig{
					{
						Port: ptr.Int(3000),
					},
				},
			},
		},
	}

	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			deployConfig,
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: &map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: ptr.String("nginx"),
			},
		},
	}
	configutil.SetFakeConfig(testConfig)

	dockerClient, err := docker.NewClient(testConfig, true)
	if err != nil {
		t.Fatalf("Error creating docker client: %v", err)
	}

	// Get image tag
	imageTag, err := randutil.GenerateRandomString(7)
	if err != nil {
		t.Fatalf("Generating imageTag failed: %v", err)
	}

	// 2. Build image
	// 3. Don't push image
	imageName := "testimage"
	network := "someNetwork"
	buildArgs := make(map[string]*string)
	imageConfig := &latest.ImageConfig{
		Image: &imageName,
		Build: &latest.BuildConfig{
			Docker: &latest.DockerConfig{
				Options: &latest.BuildOptions{
					BuildArgs: &buildArgs,
					Network: &network,
				},
			},
		},
	}
	imageBuilder, err := NewBuilder(testConfig, dockerClient, imageName, imageConfig, imageTag, true, true)
	if err != nil {
		t.Fatalf("Builder creation failed: %v", err)
	}

	err = imageBuilder.BuildImage(dir, "Dockerfile", nil, log.GetInstance())
	if err != nil {
		t.Fatalf("Image building failed: %v", err)
	} 

}

func TestDockerbuildWithEntryppointOverride(t *testing.T) {
	t.Skip("For some reason there's a problem with the coverage in this package")

	// 1. Write test dockerfile and context to a temp folder
	dir, err := ioutil.TempDir("", "testDocker")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	copy.Copy("./../../../../examples/minikube", dir)

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// 4. Cleanup temp folder
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	deployConfig := &latest.DeploymentConfig{
		Name: ptr.String("test-deployment"),
		Component: &latest.ComponentConfig{
			Containers: &[]*latest.ContainerConfig{
				{
					Image: ptr.String("nginx"),
				},
			},
			Service: &latest.ServiceConfig{
				Ports: &[]*latest.ServicePortConfig{
					{
						Port: ptr.Int(3000),
					},
				},
			},
		},
	}

	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			deployConfig,
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: &map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: ptr.String("nginx"),
			},
		},
	}
	configutil.SetFakeConfig(testConfig)

	dockerClient, err := docker.NewClient(testConfig, true)
	if err != nil {
		t.Fatalf("Error creating docker client: %v", err)
	}

	// Get image tag
	imageTag, err := randutil.GenerateRandomString(7)
	if err != nil {
		t.Fatalf("Generating imageTag failed: %v", err)
	}

	// 2. Build image with entrypoint override (see parameter entrypoint in BuildImage)
	// 3. Don't push image
	imageName := "testimage"
	imageConfig := &latest.ImageConfig{
		Image: &imageName,
	}
	imageBuilder, err := NewBuilder(testConfig, dockerClient, imageName, imageConfig, imageTag, true, true)
	if err != nil {
		t.Fatalf("Builder creation failed: %v", err)
	}

	entrypoint := make([]*string, 1)
	entryString := "node index.js"
	entrypoint[0] = &entryString
	err = imageBuilder.BuildImage(dir, "Dockerfile", &entrypoint, log.GetInstance())
	if err != nil {
		t.Fatalf("Image building failed: %v", err)
	}
}
