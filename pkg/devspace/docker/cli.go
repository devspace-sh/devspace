package docker

import (
	"github.com/loft-sh/devspace/pkg/util/log"
	"io"
	"os"
	"os/exec"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
)

// ImageBuildCLI builds an image with the docker cli
func (c *client) ImageBuildCLI(useBuildKit bool, context io.Reader, writer io.Writer, additionalArgs []string, options dockertypes.ImageBuildOptions, log log.Logger) error {
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

	for _, arg := range additionalArgs {
		args = append(args, arg)
	}

	args = append(args, "-")

	log.Infof("Execute docker cli command with: docker %s", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	cmd.Env = os.Environ()
	if useBuildKit {
		cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=1")
	}
	if c.minikubeEnv != nil {
		for k, v := range c.minikubeEnv {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	cmd.Stdin = context
	cmd.Stdout = writer
	cmd.Stderr = writer

	return cmd.Run()
}
