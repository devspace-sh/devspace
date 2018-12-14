package v1

//DevSpaceConfig defines the devspace deployment
type DevSpaceConfig struct {
	Terminal    *Terminal                `yaml:"terminal"`
	AutoReload  *AutoReloadPathsConfig   `yaml:"autoReload,omitempty"`
	Services    *[]*ServiceConfig        `yaml:"services,omitempty"`
	Deployments *[]*DeploymentConfig     `yaml:"deployments,omitempty"`
	Ports       *[]*PortForwardingConfig `yaml:"ports"`
	Sync        *[]*SyncConfig           `yaml:"sync"`
}

// AutoReloadPathsConfig defines the struct for auto reloading devspace with additional paths
type AutoReloadPathsConfig struct {
	Paths *[]*string `yaml:"paths,omitempty"`
}

// ServiceConfig defines the ports for a port forwarding to a DevSpace
type ServiceConfig struct {
	Name          *string             `yaml:"name,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector"`
	ContainerName *string             `yaml:"containerName"`
}

// PortForwardingConfig defines the ports for a port forwarding to a DevSpace
type PortForwardingConfig struct {
	Service       *string             `yaml:"service,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector"`
	PortMappings  *[]*PortMapping     `yaml:"portMappings"`
}

// PortMapping defines the ports for a PortMapping
type PortMapping struct {
	LocalPort   *int    `yaml:"localPort"`
	RemotePort  *int    `yaml:"remotePort"`
	BindAddress *string `yaml:"bindAddress"`
}

// SyncConfig defines the paths for a SyncFolder
type SyncConfig struct {
	Service              *string             `yaml:"service,omitempty"`
	Namespace            *string             `yaml:"namespace,omitempty"`
	LabelSelector        *map[string]*string `yaml:"labelSelector"`
	ContainerName        *string             `yaml:"containerName,omitempty"`
	LocalSubPath         *string             `yaml:"localSubPath"`
	ContainerPath        *string             `yaml:"containerPath"`
	ExcludePaths         *[]string           `yaml:"excludePaths"`
	DownloadExcludePaths *[]string           `yaml:"downloadExcludePaths"`
	UploadExcludePaths   *[]string           `yaml:"uploadExcludePaths"`
	BandwidthLimits      *BandwidthLimits    `yaml:"bandwidthLimits,omitempty"`
}

// BandwidthLimits defines the struct for specifying the sync bandwidth limits
type BandwidthLimits struct {
	Download *int64 `yaml:"download,omitempty"`
	Upload   *int64 `yaml:"upload,omitempty"`
}
