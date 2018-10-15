package v1

//DevSpaceConfig defines the devspace deployment
type DevSpaceConfig struct {
	Terminal       *Terminal                `yaml:"terminal"`
	Deployments    *[]*DeploymentConfig     `yaml:"deployments,omitempty"`
	PortForwarding *[]*PortForwardingConfig `yaml:"ports"`
	Sync           *[]*SyncConfig           `yaml:"sync"`
}

// PortForwardingConfig defines the ports for a port forwarding to a DevSpace
type PortForwardingConfig struct {
	Namespace     *string             `yaml:"namespace"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector"`
	PortMappings  *[]*PortMapping     `yaml:"portMappings"`
}

// PortMapping defines the ports for a PortMapping
type PortMapping struct {
	LocalPort  *int `yaml:"localPort"`
	RemotePort *int `yaml:"remotePort"`
}

// SyncConfig defines the paths for a SyncFolder
type SyncConfig struct {
	Namespace            *string             `yaml:"namespace"`
	ResourceType         *string             `yaml:"resourceType,omitempty"`
	LabelSelector        *map[string]*string `yaml:"labelSelector"`
	LocalSubPath         *string             `yaml:"localSubPath"`
	ContainerPath        *string             `yaml:"containerPath"`
	ContainerName        *string             `yaml:"containerName,omitempty"`
	ExcludePaths         *[]string           `yaml:"excludePaths"`
	DownloadExcludePaths *[]string           `yaml:"downloadExcludePaths"`
	UploadExcludePaths   *[]string           `yaml:"uploadExcludePaths"`
}
