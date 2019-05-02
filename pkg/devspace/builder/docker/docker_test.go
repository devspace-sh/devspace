package docker

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/otiai10/copy"
)

func TestDockerBuild(t *testing.T) {
	t.Log()

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

	dockerClient, err := docker.NewClient(true)
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
	imageBuilder, err := NewBuilder(dockerClient, "testimage", imageTag)
	if err != nil {
		t.Fatalf("Builder creation failed: %v", err)
	}

	err = imageBuilder.BuildImage(dir, "Dockerfile", nil, nil)
	if err != nil {
		t.Fatalf("Image building failed: %v", err)
	}

}

func TestDockerbuildWithEntryppointOverrid(t *testing.T) {
	t.Log()

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

	dockerClient, err := docker.NewClient(true)
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
	imageBuilder, err := NewBuilder(dockerClient, "testimage", imageTag)
	if err != nil {
		t.Fatalf("Builder creation failed: %v", err)
	}

	entrypoint := make([]*string, 1)
	entryString := "node index.js"
	entrypoint[0] = &entryString
	err = imageBuilder.BuildImage(dir, "Dockerfile", nil, &entrypoint)
	if err != nil {
		t.Fatalf("Image building failed: %v", err)
	}
}
