package v1

// Terminal describes the terminal options
type Terminal struct {
	ResourceType  *string             `yaml:"resourceType"`
	LabelSelector *map[string]*string `yaml:"labelSelector"`
	ContainerName *string             `yaml:"containerName"`
	Command       *[]*string          `yaml:"command"`
}
