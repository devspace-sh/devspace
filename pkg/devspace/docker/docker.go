package docker

import (
	"os"
	"os/exec"

	"context"
)

// BuildImage builds a dockerimage with the docker cli
func BuildImage(dockerfilePath, buildtag string, buildArgs []string) error {
	ctx := context.Background()
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	dockerArgs := []string{"build", cwd, "--file", dockerfilePath, "-t", buildtag}
	dockerArgs = append(dockerArgs, buildArgs...)

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	// TODO: Change output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	err = cmd.Run()

	if err != nil {
		return err
	}

	return nil
}

// PushImage pushes an image to the specified registry
func PushImage(buildtag string) error {
	ctx := context.Background()
	dockerArgs := []string{"push", buildtag}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	// TODO: Change output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	err := cmd.Run()

	if err != nil {
		return err
	}

	return nil
}
