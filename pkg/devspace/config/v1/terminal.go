package v1

// Terminal describes the terminal options
type Terminal struct {
	ContainerName *string    `yaml:"containerName"`
	Command       *[]*string `yaml:"shell"`
}
