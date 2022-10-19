package registry

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

var (
	RegistryName           = "registry"
	RegistryImage          = "registry:2.8.1"
	RegistryPort           = 5000
	RegistryDefaultStorage = "5Gi"
)

type Options struct {
	Name             string
	Namespace        string
	Image            string
	Port             int
	StorageEnabled   bool
	StorageSize      string
	StorageClassName string
}

func NewDefaultOptions() Options {
	return Options{
		Name:             RegistryName,
		Namespace:        "",
		Image:            RegistryImage,
		Port:             RegistryPort,
		StorageEnabled:   false,
		StorageSize:      RegistryDefaultStorage,
		StorageClassName: "",
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
		newOptions.Image = image
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

func (o Options) WithLocalRegistryConfig(config *latest.LocalRegistryConfig) Options {
	newOptions := o
	if config != nil {
		newOptions = newOptions.
			WithName(config.Name).
			WithNamespace(config.Namespace).
			WithImage(config.Image).
			WithPort(config.Port)

		if config.Persistence != nil && config.Persistence.Enabled != nil && *config.Persistence.Enabled {
			newOptions = newOptions.
				EnableStorage().
				WithStorageClassName(config.Persistence.StorageClassName).
				WithStorageSize(config.Persistence.Size)
		}
	}
	return newOptions
}
