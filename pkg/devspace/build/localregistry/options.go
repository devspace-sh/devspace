package localregistry

import (
	"fmt"
	"path"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	RegistryName           = "registry"
	RegistryImage          = "registry:2.8.1"
	BuildKitImage          = "moby/buildkit:master-rootless"
	RegistryPort           = 5000
	RegistryDefaultStorage = "5Gi"
)

type Options struct {
	Name             string
	Namespace        string
	LocalBuild       bool
	RegistryImage    string
	BuildKitImage    string
	Port             int
	StorageEnabled   bool
	StorageSize      string
	StorageClassName string
	Resources        *corev1.ResourceRequirements
	Annotations      map[string]string
}

func getID(o Options) string {
	return path.Join(o.Namespace, o.Name)
}

func NewDefaultOptions() Options {
	return Options{
		Name:             RegistryName,
		Namespace:        "",
		LocalBuild:       false,
		RegistryImage:    RegistryImage,
		BuildKitImage:    BuildKitImage,
		Port:             RegistryPort,
		StorageEnabled:   false,
		StorageSize:      RegistryDefaultStorage,
		StorageClassName: "",
		Resources:        nil,
		Annotations:      map[string]string{},
	}
}

func (o Options) WithName(name string) Options {
	newOptions := o
	if name != "" {
		newOptions.Name = name
	}
	return newOptions
}

func (o Options) WithNamespace(namespace string) Options {
	newOptions := o
	if namespace != "" {
		newOptions.Namespace = namespace
	}
	return newOptions
}

func (o Options) WithImage(image string) Options {
	newOptions := o
	if image != "" {
		newOptions.RegistryImage = image
	}
	return newOptions
}

func (o Options) WithBuildKitImage(image string) Options {
	newOptions := o
	if image != "" {
		newOptions.BuildKitImage = image
	}
	return newOptions
}

func (o Options) WithPort(port *int) Options {
	newOptions := o
	if port != nil {
		newOptions.Port = *port
	}
	return newOptions
}

func (o Options) WithLocalBuild(localbuild bool) Options {
	newOptions := o
	newOptions.LocalBuild = localbuild

	return newOptions
}

func (o Options) EnableStorage() Options {
	newOptions := o
	newOptions.StorageEnabled = true
	return newOptions
}

func (o Options) WithStorageClassName(storageClassName string) Options {
	newOptions := o
	if storageClassName != "" {
		newOptions.StorageClassName = storageClassName
	}
	return newOptions
}

func (o Options) WithStorageSize(storageSize string) Options {
	newOptions := o
	if storageSize != "" {
		newOptions.StorageSize = storageSize
	}
	return newOptions
}

func (o Options) WithResources(resources *latest.PodResources) Options {
	if resources == nil {
		return o
	}

	// helper converts a map[string]string -> corev1.ResourceList
	toList := func(src map[string]string) (corev1.ResourceList, error) {
		if len(src) == 0 {
			return nil, nil
		}
		dst := corev1.ResourceList{}
		for k, v := range src {
			q, err := resource.ParseQuantity(v)
			if err != nil {
				return nil, fmt.Errorf("invalid quantity %q for %s: %w", v, k, err)
			}
			dst[corev1.ResourceName(k)] = q
		}
		return dst, nil
	}

	reqs, err := toList(resources.Requests)
	if reqs == nil || err != nil {
		return o
	}
	lims, err := toList(resources.Limits)
	if lims == nil || err != nil {
		return o
	}

	o.Resources = &corev1.ResourceRequirements{
		Requests: reqs,
		Limits:   lims,
	}

	return o
}

func (o Options) WithAnnotations(annotations map[string]string) Options {
	newOptions := o
	if annotations != nil {
		newOptions.Annotations = annotations
	}
	return newOptions
}

func (o Options) WithLocalRegistryConfig(config *latest.LocalRegistryConfig) Options {
	newOptions := o
	if config != nil {
		newOptions = newOptions.
			WithName(config.Name).
			WithNamespace(config.Namespace).
			WithImage(config.Image).
			WithBuildKitImage(config.BuildKitImage).
			WithPort(config.Port).
			WithResources(config.Resources).
			WithAnnotations(config.Annotations).
			WithLocalBuild(config.LocalBuild)

		if config.Persistence != nil && config.Persistence.Enabled != nil &&
			*config.Persistence.Enabled {
			newOptions = newOptions.
				EnableStorage().
				WithStorageClassName(config.Persistence.StorageClassName).
				WithStorageSize(config.Persistence.Size)
		}
	}
	return newOptions
}
