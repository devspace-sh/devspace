package build

import (
	"context"
	"errors"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/new/framework"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/onsi/ginkgo"
)

var _ = DevSpaceDescribe("build", func() {

	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var f factory.Factory

	// create logger
	var log log.Logger

	// create context
	ctx := context.Background()

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()
	})

	ginkgo.It("should build dockerfile with docker", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		buildCmd := &cmd.BuildCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
		}

		err = buildCmd.Run(f, nil, nil)
		framework.ExpectNoError(err)

		devspaceDockerClient, err := docker.NewClient(log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerApiClient()

		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		for _, image := range imageList {
			if image.RepoTags[0] == "my-docker-username/helloworld:latest" {
				err = nil
				break
			} else {
				err = errors.New("image not found")
			}
		}
		framework.ExpectNoError(err)
	})

	ginkgo.It("should build dockerfile with buildkit", func() {
		// TODO
	})

	ginkgo.It("should build dockerfile with kaniko", func() {
		// TODO
	})

	ginkgo.It("should build dockerfile with custom builder", func() {
		// TODO
	})
})
