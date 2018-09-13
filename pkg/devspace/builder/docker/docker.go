package docker

import (
	"os"

	"context"

	"github.com/docker/docker/pkg/term"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"

	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/pkg/errors"

	"github.com/docker/docker/pkg/jsonmessage"
)

var isMinikubeVar *bool

// Builder holds the necessary information to build and push docker images
type Builder struct {
	RegistryURL string
	ImageName   string
	ImageTag    string

	imageURL string
	client   client.CommonAPIClient
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(registryURL, imageName, imageTag string, preferMinikube bool) (*Builder, error) {
	var cli client.CommonAPIClient
	var err error

	if preferMinikube {
		cli, err = newDockerClientFromMinikube()
	}
	if preferMinikube == false || err != nil {
		cli, err = newDockerClientFromEnvironment()

		if err != nil {
			return nil, err
		}
	}

	return &Builder{
		RegistryURL: registryURL,
		ImageName:   imageName,
		ImageTag:    imageTag,
		imageURL:    registryURL + "/" + imageName + ":" + imageTag,
		client:      cli,
	}, nil
}

// BuildImage builds a dockerimage with the docker cli
func (b *Builder) BuildImage(contextPath, dockerfilePath string, options *types.ImageBuildOptions) error {
	if options == nil {
		options = &types.ImageBuildOptions{}
	}

	ctx := context.Background()
	contextDir, relDockerfile, err := build.GetContextFromLocalDir(contextPath, dockerfilePath)
	if err != nil {
		return err
	}

	excludes, err := build.ReadDockerignore(contextDir)
	if err != nil {
		return err
	}

	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		return errors.Errorf("error checking context: '%s'.", err)
	}

	// And canonicalize dockerfile name to a platform-independent one
	authConfigs, _ := getAllAuthConfigs()
	relDockerfile, err = archive.CanonicalTarNameForPath(relDockerfile)
	if err != nil {
		return err
	}

	excludes = build.TrimBuildFilesFromExcludes(excludes, relDockerfile, false)
	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.IDPair{UID: 0, GID: 0},
	})
	if err != nil {
		return err
	}

	// Setup an upload progress bar
	progressOutput := streamformatter.NewProgressOutput(log.GetInstance())
	body := progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")
	response, err := b.client.ImageBuild(ctx, body, types.ImageBuildOptions{
		Tags:        []string{b.imageURL},
		Dockerfile:  relDockerfile,
		BuildArgs:   options.BuildArgs,
		AuthConfigs: authConfigs,
	})
	if err != nil {
		return errors.Wrap(err, "docker build")
	}
	defer response.Body.Close()

	fd, _ := term.GetFdInfo(os.Stdout)
	err = jsonmessage.DisplayJSONMessagesStream(response.Body, log.GetInstance(), fd, false, nil)
	if err != nil {
		return err
	}

	return nil
}

// Authenticate authenticates the cli with a remote registry
func (b *Builder) Authenticate(user, password string) error {
	return nil
}

// PushImage pushes an image to the specified registry
func (b *Builder) PushImage() error {
	/*if isMinikube() {
		err := pushImageMinikube(buildtag)

		if err == nil {
			return nil
		}

		// Fallback to normal docker cli if minikube failed
	}

	ctx := context.Background()
	dockerArgs := []string{"push", buildtag}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	cmd.Stdout = log.GetInstance()
	cmd.Stderr = log.GetInstance()

	err := cmd.Run()

	if err != nil {
		return err
	}*/

	return nil
}
