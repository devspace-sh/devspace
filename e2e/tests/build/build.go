package build

import (
	"context"
	"errors"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
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

	// Test cases:

	ginkgo.It("should build dockerfile with docker", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.BuildCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
		}
		err = buildCmd.Run(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
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
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/buildkit")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.BuildCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
		}
		err = buildCmd.Run(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		for _, image := range imageList {
			if image.RepoTags[0] == "my-docker-username/helloworld-buildkit:latest" {
				err = nil
				break
			} else {
				err = errors.New("image not found")
			}
		}
		framework.ExpectNoError(err)
	})

	ginkgo.It("should build dockerfile with kaniko", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/kaniko")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.BuildCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
		}
		err = buildCmd.Run(f)
		framework.ExpectNoError(err)
	})

	ginkgo.It("should build dockerfile with custom builder", func() {
		// TODO
	})
})
