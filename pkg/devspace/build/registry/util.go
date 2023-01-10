package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/kaniko"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dockerclient "github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	corev1 "k8s.io/api/core/v1"
)

func HasPushPermission(image *latest.Image) bool {
	ref, err := name.ParseReference(image.Image)
	if err != nil {
		panic(err)
	}

	pushErr := remote.CheckPushPermission(ref, authn.DefaultKeychain, http.DefaultTransport)
	return pushErr == nil
}

func IsLocalRegistryFallback(config *latest.Config) bool {
	return config.LocalRegistry == nil || (config.LocalRegistry != nil && config.LocalRegistry.Enabled == nil)
}

func IsLocalRegistryEnabled(config *latest.Config) bool {
	return config.LocalRegistry != nil && config.LocalRegistry.Enabled != nil && *config.LocalRegistry.Enabled
}

func GetServicePort(service *corev1.Service) *corev1.ServicePort {
	for _, port := range service.Spec.Ports {
		if port.Name == "registry" {
			return &port
		}
	}
	return nil
}

func IsImageAvailableRemotely(ctx context.Context, imageName string) (bool, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return false, err
	}

	image, err := remote.Image(
		ref,
		remote.WithContext(ctx),
		remote.WithTransport(remote.DefaultTransport),
	)
	if err != nil {
		transportError, ok := err.(*transport.Error)
		if ok && transportError.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}

	return image != nil, nil
}

func CopyImageToRemote(ctx context.Context, client dockerclient.Client, imageName string, writer io.Writer) error {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return err
	}

	image, err := daemon.Image(ref, daemon.WithContext(ctx), daemon.WithClient(client.DockerAPIClient()))
	if err != nil {
		return err
	}

	progressChan := make(chan v1.Update, 200)
	errChan := make(chan error, 1)
	go func() {
		errChan <- remote.Write(
			ref,
			image,
			remote.WithContext(ctx),
			remote.WithTransport(remote.DefaultTransport),
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
			ID:     ref.Identifier(),
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

func UseLocalRegistry(client kubectl.Client, config *latest.Config, imageConfig *latest.Image, imageBuilder builder.Interface, skipPush bool) bool {
	if skipPush {
		return false
	} else if client == nil {
		return false
	}

	// check if node architecture equals our architecture
	if runtime.GOARCH != "amd64" && client.KubeClient() != nil {
		nodes, err := client.KubeClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return false
		} else if len(nodes.Items) != 1 {
			return false
		} else if nodes.Items[0].Labels == nil || nodes.Items[0].Labels["kubernetes.io/arch"] != runtime.GOARCH {
			return false
		}
	}

	// check if builder is kaniko
	if imageBuilder != nil {
		_, ok := imageBuilder.(*kaniko.Builder)
		if ok {
			return false
		}
	}

	// check if image looks weird like localhost / cluster.local
	if imageConfig != nil {
		if imageConfig.Kaniko != nil {
			return false
		} else if imageConfig.Custom != nil {
			return false
		} else if imageConfig.BuildKit != nil && imageConfig.BuildKit.InCluster != nil {
			return false
		}

		imageWithoutPort := strings.Split(imageConfig.Image, ":")[0]
		if imageWithoutPort == "" || imageWithoutPort == "localhost" || imageWithoutPort == "127.0.0.1" || strings.HasSuffix(imageWithoutPort, ".local") || strings.HasSuffix(imageWithoutPort, ".localhost") {
			return false
		}
	}

	// check if fallback
	if !IsLocalRegistryFallback(config) {
		return IsLocalRegistryEnabled(config)
	}

	// Determine if this is a vcluster
	context := client.CurrentContext()
	isVClusterContext := strings.Contains(context, "vcluster_")

	// Determine if this is a local kubernetes cluster
	isLocalKubernetes := kubectl.IsLocalKubernetes(context)
	return !isLocalKubernetes && !(isVClusterContext && isLocalKubernetes)
}
