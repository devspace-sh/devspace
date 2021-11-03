package build

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

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
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/custom_build")
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
			if image.RepoTags[0] == "my-docker-username/helloworld-custom-build:latest" {
				err = nil
				break
			} else {
				err = errors.New("image not found")
			}
		}
		framework.ExpectNoError(err)
	})

	ginkgo.It("should ignore files from Dockerfile.dockerignore", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/dockerignore")
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
		imageName := "my-docker-username/helloworld-dokcerignore:latest"
		for _, image := range imageList {
			if image.RepoTags[0] == imageName {
				err = nil
				break
			} else {
				err = errors.New("image not found")
			}
		}
		framework.ExpectNoError(err)

		resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
			Image: imageName,
			Cmd:   []string{"/bin/ls", "./build"},
			Tty:   false,
		}, nil, nil, nil, "")
		framework.ExpectNoError(err)

		err = dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
		framework.ExpectNoError(err)

		statusCh, errCh := dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			framework.ExpectNoError(err)
		case <-statusCh:
		}

		out, err := dockerClient.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
		framework.ExpectNoError(err)

		stdout := &bytes.Buffer{}
		_, err = io.Copy(stdout, out)
		framework.ExpectNoError(err)

		err = stdoutContains(stdout.String(), "bar.txt")
		framework.ExpectError(err)

		err = stdoutContains(stdout.String(), "foo.txt")
		framework.ExpectError(err)

		fmt.Println(stdout.String())
	})
})

func stdoutContains(stdout, content string) error {
	if strings.Contains(stdout, content) {
		return nil
	}
	return fmt.Errorf("%s found in output", content)
}
