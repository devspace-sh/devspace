package docker

import (
	"io"
	"os"
	"os/exec"

	dockertypes "github.com/docker/docker/api/types"
)

// ImageBuildCLI builds an image with the docker cli
func (c *client) ImageBuildCLI(useBuildkit bool, context io.Reader, writer io.Writer, options dockertypes.ImageBuildOptions) error {
	args := []string{"build"}

	if options.BuildArgs != nil {
		for k, v := range options.BuildArgs {
			if v == nil {
				continue
			}

			args = append(args, "--build-arg", k+"="+*v)
		}
	}
	if options.NetworkMode != "" {
		args = append(args, "--network", options.NetworkMode)
	}
	for _, tag := range options.Tags {
		args = append(args, "--tag", tag)
	}

	if options.Dockerfile != "" {
		args = append(args, "--file", options.Dockerfile)
	}
	if options.Target != "" {
		args = append(args, "--target", options.Target)
	}

	args = append(args, "-")

	cmd := exec.Command("docker", args...)
	if useBuildkit {
		cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	}

	cmd.Stdin = context
	cmd.Stdout = writer
	cmd.Stderr = writer

	return cmd.Run()
}
