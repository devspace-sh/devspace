package v1

// Version is the current api version
const Version string = "v1"

//Config defines the configuration
type Config struct {
	Version    *string                     `yaml:"version"`
	DevSpace   *DevSpaceConfig             `yaml:"devSpace,omitempty"`
	Images     *map[string]*ImageConfig    `yaml:"images,omitempty"`
	Registries *map[string]*RegistryConfig `yaml:"registries,omitempty"`
	Cluster    *Cluster                    `yaml:"cluster,omitempty"`
	Services   *ServiceConfig              `yaml:"services,omitempty"`
}

//Release defines running version of a project
type Release struct {
	Name      *string                      `yaml:"name"`
	Namespace *string                      `yaml:"namespace"`
	Values    *map[interface{}]interface{} `yaml:"values,omitempty"`
}
