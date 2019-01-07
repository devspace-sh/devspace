package v1

//DevSpaceConfig defines the devspace deployment
type DevSpaceConfig struct {
	Terminal    *Terminal                `yaml:"terminal,omitempty"`
	AutoReload  *AutoReloadPathsConfig   `yaml:"autoReload,omitempty"`
	Services    *[]*ServiceConfig        `yaml:"services,omitempty"`
	Deployments *[]*DeploymentConfig     `yaml:"deployments,omitempty"`
	Ports       *[]*PortForwardingConfig `yaml:"ports,omitempty"`
	Sync        *[]*SyncConfig           `yaml:"sync,omitempty"`
}

// AutoReloadPathsConfig defines the struct for auto reloading devspace with additional paths
type AutoReloadPathsConfig struct {
	Paths *[]*string `yaml:"paths,omitempty"`
}

// ServiceConfig defines the kubernetes services that belong to the devspace
type ServiceConfig struct {
	Name          *string             `yaml:"name,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector"`
	ContainerName *string             `yaml:"containerName,omitempty"`
}

// PortForwardingConfig defines the ports for a port forwarding to a DevSpace
type PortForwardingConfig struct {
	Service       *string             `yaml:"service,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector,omitempty"`
	PortMappings  *[]*PortMapping     `yaml:"portMappings"`
}

// PortMapping defines the ports for a PortMapping
type PortMapping struct {
	LocalPort   *int    `yaml:"localPort"`
	RemotePort  *int    `yaml:"remotePort"`
	BindAddress *string `yaml:"bindAddress,omitempty"`
}

// SyncConfig defines the paths for a SyncFolder
type SyncConfig struct {
	Service              *string             `yaml:"service,omitempty"`
	Namespace            *string             `yaml:"namespace,omitempty"`
	LabelSelector        *map[string]*string `yaml:"labelSelector,omitempty"`
	ContainerName        *string             `yaml:"containerName,omitempty"`
	LocalSubPath         *string             `yaml:"localSubPath,omitempty"`
	ContainerPath        *string             `yaml:"containerPath,omitempty"`
	ExcludePaths         *[]string           `yaml:"excludePaths,omitempty"`
	DownloadExcludePaths *[]string           `yaml:"downloadExcludePaths,omitempty"`
	UploadExcludePaths   *[]string           `yaml:"uploadExcludePaths,omitempty"`
	BandwidthLimits      *BandwidthLimits    `yaml:"bandwidthLimits,omitempty"`
}

// BandwidthLimits defines the struct for specifying the sync bandwidth limits
type BandwidthLimits struct {
	Download *int64 `yaml:"download,omitempty"`
	Upload   *int64 `yaml:"upload,omitempty"`
}
