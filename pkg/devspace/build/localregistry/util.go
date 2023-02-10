package localregistry

import (
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	corev1 "k8s.io/api/core/v1"
)

func HasPushPermission(image *latest.Image) bool {
	ref, err := name.ParseReference(image.Image)
	if err != nil {
		panic(err)
	}

	pushErr := remote.CheckPushPermission(ref, authn.DefaultKeychain, http.DefaultTransport)

	if isInsecureRegistry(pushErr) {
		// Retry with insecure registry
		ref, err := name.ParseReference(image.Image, name.Insecure)
		if err != nil {
			panic(err)
		}

		pushErr = remote.CheckPushPermission(ref, authn.DefaultKeychain, http.DefaultTransport)
	}

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

func UseLocalRegistry(client kubectl.Client, config *latest.Config, imageConfig *latest.Image, skipPush bool) bool {
	if skipPush {
		return false
	} else if client == nil {
		return false
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
	}

	// check if fallback
	if IsLocalRegistryEnabled(config) {
		return true
	} else if !IsLocalRegistryFallback(config) {
		return false
	}

	// Determine if this is a vcluster
	context := client.CurrentContext()
	isVClusterContext := strings.Contains(context, "vcluster_")

	// Determine if this is a local kubernetes cluster
	isLocalKubernetes := kubectl.IsLocalKubernetes(client)
	return !isLocalKubernetes && !(isVClusterContext && isLocalKubernetes)
}

func IsImageAvailableInLocalRegistry(ctx devspacecontext.Context, registryPod *corev1.Pod, imageName string) (bool, error) {
	ref, err := name.NewTag(imageName)
	if err != nil {
		return false, err
	}

	// build file path
	filePath := fmt.Sprintf("/var/lib/registry/docker/registry/v2/repositories/%s/_manifests/tags/%s", ref.RepositoryStr(), ref.TagStr())
	out, err := ctx.KubeClient().ExecBufferedCombined(ctx.Context(), registryPod, "registry", []string{"ls", filePath}, nil)
	if err != nil {
		ctx.Log().Debugf("Error retrieving tag: %s %v", string(out), err)
		return false, nil
	}

	return true, nil
}

func isInsecureRegistry(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "http: server gave HTTP response to HTTPS client")
}
