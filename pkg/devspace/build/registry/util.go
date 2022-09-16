package registry

import (
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
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

func IsLocalRegistryDisabled(image *latest.Image) bool {
	return image.LocalRegistry != nil && image.LocalRegistry.Disable
}

func IsLocalRegistrySupported(image *latest.Image) bool {
	if image.Custom != nil {
		return false
	}

	if image.Kaniko != nil {
		return false
	}

	if image.Docker == nil && image.Kaniko == nil {
		return true
	}

	if image.BuildKit != nil {
		return true
	}

	return false
}

func GetNodePort(service *corev1.Service) int32 {
	for _, port := range service.Spec.Ports {
		if port.Name == "registry" {
			return port.NodePort
		}
	}
	return 0
}
