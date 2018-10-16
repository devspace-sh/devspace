package v1

// Version is the current api version
const Version string = "v1"

// Config defines the configuration
type Config struct {
	Version          *string                     `yaml:"version"`
	DevSpace         *DevSpaceConfig             `yaml:"devSpace,omitempty"`
	Images           *map[string]*ImageConfig    `yaml:"images,omitempty"`
	Registries       *map[string]*RegistryConfig `yaml:"registries,omitempty"`
	Cluster          *Cluster                    `yaml:"cluster,omitempty"`
	Tiller           *TillerConfig               `yaml:"tiller,omitempty"`
	InternalRegistry *InternalRegistryConfig     `yaml:"internalRegistry,omitempty"`
}

// TillerConfig defines the tiller service
type TillerConfig struct {
	Namespace *string `yaml:"namespace,omitempty"`
}

// InternalRegistryConfig defines the internal registry config options
type InternalRegistryConfig struct {
	Deploy    *bool   `yaml:"deploy,omitempty"`
	Namespace *string `yaml:"namespace,omitempty"`
}
