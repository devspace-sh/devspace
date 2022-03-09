package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/onsi/ginkgo"
)

const IMAGE = "username/app:latest"

var _ = DevSpaceDescribe("build", func() {

	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var f factory.Factory

	// create logger
	log := log.GetBaseInstance()

	// create context
	ctx := context.Background()

	// create devspace docker client to access docker APIs
	devspaceDockerClient, err := docker.NewClient(log)
	framework.ExpectNoError(err)
	dockerClient := devspaceDockerClient.DockerAPIClient()

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()
	})

	// Test cases:
	ginkgo.FIt("should build dockerfile with docker", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
		log.Info("Running test for old config")
		log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")

		err = test(ctx, f, "", dockerClient)
		framework.ExpectNoError(err)

		// remove image
		_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
		framework.ExpectNoError(err)

		log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
		log.Info("Running test for v6 config")
		log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")

		err = test(ctx, f, "devspace-v6.yaml", dockerClient)
		framework.ExpectNoError(err)

		_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
		framework.ExpectNoError(err)

	})

	ginkgo.FIt("should build dockerfile with buildkit", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/buildkit")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
		log.Info("Running test for old config")
		log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")

		err = test(ctx, f, "", dockerClient)
		framework.ExpectNoError(err)

		_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
		framework.ExpectNoError(err)

		log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
		log.Info("Running test for v6 config")
		log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")

		err = test(ctx, f, "devspace-v6.yaml", dockerClient)
		framework.ExpectNoError(err)

		_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
		framework.ExpectNoError(err)

	})

	ginkgo.FIt("should build dockerfile with kaniko", func() {
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

		log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
		log.Info("Running test for v6")
		log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")

		buildCmd = &cmd.BuildCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				ConfigPath: "devspace-v6.yaml",
			},
			SkipPush: true,
		}
		err = buildCmd.Run(f)
		framework.ExpectNoError(err)
	})

	ginkgo.FIt("should build dockerfile with custom builder", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/custom_build")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
		log.Info("Running test for old config")
		log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")

		err = test(ctx, f, "", dockerClient)
		framework.ExpectNoError(err)

		_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
		framework.ExpectNoError(err)

		log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
		log.Info("Running test for v6 config")
		log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")

		err = test(ctx, f, "devspace-v6.yaml", dockerClient)
		framework.ExpectNoError(err)

		_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
		framework.ExpectNoError(err)

	})

	ginkgo.FIt("should ignore files from Dockerfile.dockerignore only", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/dockerignore")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		test2(ctx, f, dockerClient, log)
	})

	ginkgo.FIt("should ignore files from Dockerfile.dockerignore relative path", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/dockerignore_rel_path")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		test2(ctx, f, dockerClient, log)
	})

	ginkgo.FIt("should ignore files from outside of context Dockerfile.dockerignore", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/dockerignore_context")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
		log.Info("Running test for old config")
		log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")
		// create build command
		buildCmd := &cmd.BuildCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush: true,
		}
		err = buildCmd.Run(f)
		framework.ExpectNoError(err)

		imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		err = findImage(imageList)
		framework.ExpectNoError(err)

		resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
			Image: IMAGE,
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

		err = dockerClient.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		framework.ExpectNoError(err)

		_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
		framework.ExpectNoError(err)

		log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
		log.Info("Running test for v6 config")
		log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")

		// create build command
		buildCmd = &cmd.BuildCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				ConfigPath: "devspace-v6.yaml",
			},
			SkipPush: true,
		}
		err = buildCmd.Run(f)
		framework.ExpectNoError(err)

		imageList, err = dockerClient.ImageList(ctx, types.ImageListOptions{})
		framework.ExpectNoError(err)

		err = findImage(imageList)
		framework.ExpectNoError(err)

		resp, err = dockerClient.ContainerCreate(ctx, &container.Config{
			Image: IMAGE,
			Cmd:   []string{"/bin/ls", "./build"},
			Tty:   false,
		}, nil, nil, nil, "")
		framework.ExpectNoError(err)

		err = dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
		framework.ExpectNoError(err)

		statusCh, errCh = dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			framework.ExpectNoError(err)
		case <-statusCh:
		}

		out, err = dockerClient.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
		framework.ExpectNoError(err)

		stdout = &bytes.Buffer{}
		_, err = io.Copy(stdout, out)
		framework.ExpectNoError(err)

		err = stdoutContains(stdout.String(), "bar.txt")
		framework.ExpectNoError(err)

		err = stdoutContains(stdout.String(), "foo.txt")
		framework.ExpectError(err)

		err = dockerClient.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		framework.ExpectNoError(err)

		_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
		framework.ExpectNoError(err)

	})
})

func test(ctx context.Context, f factory.Factory, configPath string, dockerClient client.CommonAPIClient) error {
	buildCmd := &cmd.BuildCmd{
		GlobalFlags: &flags.GlobalFlags{
			NoWarn:     true,
			ConfigPath: configPath,
		},
		SkipPush: true,
	}
	err := buildCmd.Run(f)
	if err != nil {
		return err
	}
	imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return err
	}

	err = findImage(imageList)
	if err != nil {
		return err
	}
	return nil
}

func findImage(imageList []types.ImageSummary) (err error) {
Outer:
	for _, i := range imageList {
		for _, repoTag := range i.RepoTags {
			if repoTag == IMAGE {
				err = nil
				break Outer
			} else {
				err = fmt.Errorf(`image "%s" not found`, IMAGE)
			}
		}
	}

	return err
}

func test2(ctx context.Context, f factory.Factory, dockerClient client.CommonAPIClient, log log.Logger) {
	log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
	log.Info("Running test for old config")
	log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")
	// create build command
	buildCmd := &cmd.BuildCmd{
		GlobalFlags: &flags.GlobalFlags{
			NoWarn: true,
		},
		SkipPush: true,
	}
	err := buildCmd.Run(f)
	framework.ExpectNoError(err)

	imageList, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
	framework.ExpectNoError(err)

	err = findImage(imageList)
	framework.ExpectNoError(err)

	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: IMAGE,
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

	err = dockerClient.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
	framework.ExpectNoError(err)

	_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
	framework.ExpectNoError(err)

	log.WriteString(logrus.InfoLevel, "\n*********************************************************\n")
	log.Info("Running test for v6 config")
	log.WriteString(logrus.InfoLevel, "*********************************************************\n\n")

	// create build command
	buildCmd = &cmd.BuildCmd{
		GlobalFlags: &flags.GlobalFlags{
			NoWarn:     true,
			ConfigPath: "devspace-v6.yaml",
		},
		SkipPush: true,
	}
	err = buildCmd.Run(f)
	framework.ExpectNoError(err)

	imageList, err = dockerClient.ImageList(ctx, types.ImageListOptions{})
	framework.ExpectNoError(err)

	err = findImage(imageList)
	framework.ExpectNoError(err)

	resp, err = dockerClient.ContainerCreate(ctx, &container.Config{
		Image: IMAGE,
		Cmd:   []string{"/bin/ls", "./build"},
		Tty:   false,
	}, nil, nil, nil, "")
	framework.ExpectNoError(err)

	err = dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	framework.ExpectNoError(err)

	statusCh, errCh = dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		framework.ExpectNoError(err)
	case <-statusCh:
	}

	out, err = dockerClient.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	framework.ExpectNoError(err)

	stdout = &bytes.Buffer{}
	_, err = io.Copy(stdout, out)
	framework.ExpectNoError(err)

	err = stdoutContains(stdout.String(), "bar.txt")
	framework.ExpectError(err)

	err = stdoutContains(stdout.String(), "foo.txt")
	framework.ExpectError(err)

	err = dockerClient.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
	framework.ExpectNoError(err)

	_, err = dockerClient.ImageRemove(ctx, IMAGE, types.ImageRemoveOptions{Force: true})
	framework.ExpectNoError(err)
}

func stdoutContains(stdout, content string) error {
	if strings.Contains(stdout, content) {
		return nil
	}
	return fmt.Errorf("%s found in output", content)
}
