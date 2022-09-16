package registry

import "fmt"

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
		newOptions.Name = image
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

func (o Options) ID() string {
	return fmt.Sprintf("%s/%s", o.Namespace, o.Name)
}
