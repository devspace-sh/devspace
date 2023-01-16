package localregistry

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devspace/pkg/devspace/build/localregistry"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/pkg/errors"
	"io"
	"net"
	"strings"

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

	dockerConfig, err := docker.LoadDockerConfig()
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
