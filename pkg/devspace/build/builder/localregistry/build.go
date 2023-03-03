package localregistry

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devspace/pkg/devspace/build/localregistry"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/pkg/errors"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"

	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/upload/uploadprovider"
)

func RemoteBuild(ctx devspacecontext.Context, podName, namespace string, buildContext io.Reader, writer io.Writer, buildOptions *types.ImageBuildOptions) error {
	conn, err := ExecConn(ctx, namespace, podName, localregistry.BuildKitContainer, []string{"buildctl", "dial-stdio"})
	if err != nil {
		return errors.Wrap(err, "connect to buildkit pod")
	}

	// create new buildkit client
	client, err := buildkit.New(ctx.Context(), "", buildkit.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return conn, nil
	}))
	if err != nil {
		return err
	}
	defer client.Close()

	dockerConfig, err := dockerclient.LoadDockerConfig()
	if err != nil {
		return err
	}

	// stdin is context
	up := uploadprovider.New()
	options := buildkit.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"filename": buildOptions.Dockerfile,
			"target":   buildOptions.Target,
			"context":  up.Add(buildContext),
		},
		Session: []session.Attachable{up, authprovider.NewDockerAuthProvider(dockerConfig)},
		Exports: []buildkit.ExportEntry{
			{
				Type: buildkit.ExporterImage,
				Attrs: map[string]string{
					"name":           strings.Join(buildOptions.Tags, ","),
					"name-canonical": "",
					"push":           "true",
				},
			},
		},
	}

	for key, value := range buildOptions.BuildArgs {
		if value == nil {
			continue
		}
		options.FrontendAttrs["build-arg:"+key] = *value
	}

	pw, err := NewPrinter(context.TODO(), writer)
	if err != nil {
		return err
	}

	_, err = client.Solve(ctx.Context(), nil, options, pw.Status())
	if err != nil {
		return err
	}

	return nil
}

// LocalBuild builds a dockerimage with the docker cli
// contextPath is the absolute path to the context path
// dockerfilePath is the absolute path to the dockerfile WITHIN the contextPath
func LocalBuild(ctx devspacecontext.Context, contextPath, dockerfilePath string, entrypoint []string, cmd []string, b *Builder) error {
	// create context stream
	body, writer, outStream, buildOptions, err := b.helper.CreateContextStream(contextPath, dockerfilePath, entrypoint, cmd, ctx.Log())
	defer writer.Close()
	if err != nil {
		return err
	}

	dockerClient, err := dockerclient.NewClient(ctx.Context(), ctx.Log())
	if err != nil {
		return nil
	}

	// make sure to use the correct proxy configuration
	buildOptions.BuildArgs = dockerClient.ParseProxyConfig(buildOptions.BuildArgs)

	response, err := dockerClient.ImageBuild(ctx.Context(), body, *buildOptions)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	err = jsonmessage.DisplayJSONMessagesStream(response.Body, outStream, outStream.FD(), outStream.IsTerminal(), nil)
	if err != nil {
		return err
	}

	for _, tag := range buildOptions.Tags {
		ctx.Log().Info("The push refers to repository [" + tag + "]")
		err := CopyImageToRemote(ctx.Context(), dockerClient, tag, writer, b)
		if err != nil {
			return errors.Errorf("error during local registry image push: %v", err)
		}

		ctx.Log().Info("Image pushed to local registry")
	}

	return nil
}

// CopyImageToRemote will extract an image from a local registry and stream it to a remote registry
func CopyImageToRemote(ctx context.Context, client dockerclient.Client, imageName string, writer io.Writer, b *Builder) error {
	// get local registry data
	localRef, err := name.ParseReference(imageName)
	if err != nil {
		return err
	}
	// get remote registry data
	remoteRef, err := name.ParseReference(imageName, name.WithDefaultRegistry(b.localRegistry.GetRegistryURL()))
	if err != nil {
		return err
	}
	// get image data from local registry
	image, err := daemon.Image(localRef, daemon.WithContext(ctx), daemon.WithClient(client.DockerAPIClient()))
	if err != nil {
		return err
	}

	progressChan := make(chan v1.Update, 200)
	errChan := make(chan error, 1)
	// push image to remote registry
	go func() {
		errChan <- remote.Write(
			remoteRef,
			image,
			remote.WithContext(ctx),
			remote.WithProgress(progressChan),
		)
	}()

	for update := range progressChan {
		if update.Error != nil {
			return err
		}

		status := "Pushing"
		if update.Complete == update.Total {
			status = "Pushed"
		}

		jm := &jsonmessage.JSONMessage{
			ID:     localRef.Identifier(),
			Status: status,
			Progress: &jsonmessage.JSONProgress{
				Current: update.Complete,
				Total:   update.Total,
			},
		}

		_, err := fmt.Fprintf(writer, "%s %s\n", jm.Status, jm.Progress.String())
		if err != nil {
			return err
		}
	}

	return <-errChan
}
