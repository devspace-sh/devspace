package v1

// Terminal describes the terminal options
type Terminal struct {
	Disabled      *bool               `yaml:"disabled,omitempty"`
	Service       *string             `yaml:"service,omitempty"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ContainerName *string             `yaml:"containerName,omitempty"`
	Command       *[]*string          `yaml:"command,omitempty"`
}
