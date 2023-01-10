package build

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
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
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		found := false
	Outer:
		for _, image := range imageList {
			for _, tag := range image.RepoTags {
				if tag == "my-docker-username/helloworld:latest" {
					found = true
					break Outer
				}
			}
		}
		framework.ExpectEqual(found, true, "image not found in cache")
	})
	ginkgo.It("should build dockerfile with docker and skip-dependency", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker-skip-dependency")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			SkipDependency: []string{
				"fake-dep",
			},
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		found := false
	Outer:
		for _, image := range imageList {
			for _, tag := range image.RepoTags {
				if tag == "my-docker-username/helloworld:latest" {
					found = true
					break Outer
				}
			}
		}
		framework.ExpectEqual(found, true, "image not found in cache")
	})

	ginkgo.It("should build dockerfile with docker and load in kind cluster", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		found := false
	Outer:
		for _, image := range imageList {
			for _, tag := range image.RepoTags {
				if tag == "my-docker-username/helloworld:latest" {
					found = true
					break Outer
				}
			}
		}
		framework.ExpectEqual(found, true, "image not found in cache")

		var stdout, stderr bytes.Buffer
		cmd := exec.Command("kind", "load", "docker-image", "my-docker-username/helloworld:latest")
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		framework.ExpectNoError(err)
		err = stderrContains(stderr.String(), "found to be already present")
		framework.ExpectNoError(err)
	})

	ginkgo.It("should build dockerfile with docker even when KUBECONFIG is invalid", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		_ = os.Setenv("KUBECONFIG", "i-am-invalid-config")
		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		found := false
	Outer:
		for _, image := range imageList {
			for _, tag := range image.RepoTags {
				if tag == "my-docker-username/helloworld:latest" {
					found = true
					break Outer
				}
			}
		}
		framework.ExpectEqual(found, true, "image not found in cache")
		_ = os.Unsetenv("KUBECONFIG")
	})

	ginkgo.It("should not build dockerfile with kaniko when KUBECONFIG is invalid", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/kaniko")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)
		_ = os.Setenv("KUBECONFIG", "i-am-invalid-config")
		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectError(err)
		_ = os.Unsetenv("KUBECONFIG")
	})

	ginkgo.It("should build dockerfile with buildkit and load in kind cluster", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/buildkit")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		for _, image := range imageList {
			if len(image.RepoTags) > 0 && image.RepoTags[0] == "my-docker-username/helloworld-buildkit:latest" {
				err = nil
				break
			} else {
				err = errors.New("image not found")
			}
		}
		framework.ExpectNoError(err)

		var stdout, stderr bytes.Buffer
		cmd := exec.Command("kind", "load", "docker-image", "my-docker-username/helloworld-buildkit:latest")
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		framework.ExpectNoError(err)
		err = stderrContains(stderr.String(), "found to be already present")
		framework.ExpectNoError(err)
	})

	ginkgo.It("should build dockerfile with buildkit", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/buildkit")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		for _, image := range imageList {
			if len(image.RepoTags) > 0 && image.RepoTags[0] == "my-docker-username/helloworld-buildkit:latest" {
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
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)
	})

	ginkgo.It("should build dockerfile with custom builder", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/custom_build")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		for _, image := range imageList {
			if len(image.RepoTags) > 0 && image.RepoTags[0] == "my-docker-username/helloworld-custom-build:latest" {
				err = nil
				break
			} else {
				err = errors.New("image not found")
			}
		}
		framework.ExpectNoError(err)
	})

	ginkgo.It("should ignore files from Dockerfile.dockerignore only", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/dockerignore")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)
		imageName := "my-docker-username/helloworld-dockerignore:latest"
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
	})

	ginkgo.It("should ignore files from Dockerfile.dockerignore relative path", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/dockerignore_rel_path")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)
		imageName := "my-docker-username/helloworld-dockerignore-rel-path:latest"
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
	})

	ginkgo.It("should ignore files from outside of context Dockerfile.dockerignore", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/dockerignore_context")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
			Pipeline: "build",
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// create devspace docker client to access docker APIs
		devspaceDockerClient, err := docker.NewClient(context.TODO(), log)
		framework.ExpectNoError(err)

		dockerClient := devspaceDockerClient.DockerAPIClient()
		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)
		imageName := "my-docker-username/helloworld-dockerignore-context:latest"
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
		framework.ExpectNoError(err)

		err = stdoutContains(stdout.String(), "foo.txt")
		framework.ExpectError(err)
	})
})

func stdoutContains(stdout, content string) error {
	if strings.Contains(stdout, content) {
		return nil
	}
	return fmt.Errorf("%s found in output", content)
}

func stderrContains(stderr, content string) error {
	if strings.Contains(stderr, content) {
		return nil
	}
	return fmt.Errorf("%s found in output", content)
}
