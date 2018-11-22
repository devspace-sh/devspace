package v1

// Terminal describes the terminal options
type Terminal struct {
	Disabled      *bool               `yaml:"disabled,omitempty"`
	Service       *string             `yaml:"service,omitempty"`
	ResourceType  *string             `yaml:"resourceType"`
	LabelSelector *map[string]*string `yaml:"labelSelector"`
	Namespace     *string             `yaml:"namespace"`
	ContainerName *string             `yaml:"containerName"`
	Command       *[]*string          `yaml:"command"`
}
