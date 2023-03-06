package docker

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/env"
	"github.com/loft-sh/utils/pkg/command"
	"io"
	"mvdan.cc/sh/v3/expand"
	"strings"

	"github.com/loft-sh/devspace/pkg/util/log"

	dockertypes "github.com/docker/docker/api/types"
)

// ImageBuildCLI builds an image with the docker cli
func (c *client) ImageBuildCLI(ctx context.Context, workingDir string, environ expand.Environ, useBuildKit bool, context io.Reader, writer io.Writer, additionalArgs []string, options dockertypes.ImageBuildOptions, log log.Logger) error {
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

	args = append(args, additionalArgs...)
	args = append(args, "-")

	log.Infof("Execute docker cli command with: docker %s", strings.Join(args, " "))

	extraEnv := map[string]string{}
	if useBuildKit {
		extraEnv["DOCKER_BUILDKIT"] = "1"
	}
	if c.minikubeEnv != nil {
		for k, v := range c.minikubeEnv {
			extraEnv[k] = v
		}
	}

	environ = env.NewVariableEnvProvider(environ, extraEnv)
	return command.Command(ctx, workingDir, environ, writer, writer, context, "docker", args...)
}
