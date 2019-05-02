package docker

import (
	"fmt"
	"ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/covexo/devspace/pkg/devspace/builder/docker"
	"github.com/covexo/devspace/pkg/util/randutil"
	"github.com/otiai10/copy"
)

func TestDockerBuild(t *testing.T) {
	
	// @Florian
	// 1. Write test dockerfile and context to a temp folder
	// 2. Build image
	// 3. Don't push image
	// 4. Cleanup temp folder
	dir, err := ioutil.TempDir("", "testDocker")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir) // clean up

	Copy("./../../../examples/quickstart", dir.path)

	tmpfn := filepath.Join(dir, "Dockerfile")
	err = ioutil.WriteFile(tmpfn, content, 0666)
	if err != nil {
		log.Fatal(err)
	}

	dockerClient, err := dockerclient.NewClient(true)
	if err != nil {
		return nil, fmt.Errorf("Error creating docker client: %v", err)
	}

	// Get image tag
	imageTag, err := randutil.GenerateRandomString(7)
	if err != nil {
		return false, fmt.Errorf("Image building failed: %v", err)
	}

	imageBuilder, err = docker.NewBuilder(dockerClient, *imageConf.Image, imageTag)
}

func TestDockerbuildWithEntryppointOverrid(t *testing.T) {
	// @Florian
	// 1. Write test dockerfile and context to a temp folder
	// 2. Build image with entrypoint override (see parameter entrypoint in BuildImage)
	// 3. Don't push image
	// 4. Cleanup temp folder
}
