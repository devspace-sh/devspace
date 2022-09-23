package registry

import (
	"context"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
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

func IsLocalRegistryDisabled(config *latest.Config) bool {
	return config.LocalRegistry != nil && config.LocalRegistry.Disable
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

func CopyImageToRemote(ctx context.Context, imageName string) error {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return err
	}

	image, err := daemon.Image(ref, daemon.WithContext(ctx))
	if err != nil {
		return err
	}

	err = remote.Write(
		ref,
		image,
		remote.WithContext(ctx),
		remote.WithTransport(remote.DefaultTransport),
	)
	if err != nil {
		return err
	}

	return nil
}
