package v1

// Version is the current api version
const Version string = "v1"

//Config defines the configuration
type Config struct {
	Version  *string         `yaml:"version"`
	DevSpace *DevSpaceConfig `yaml:"devSpace,omitempty"`
	Image    *ImageConfig    `yaml:"image,omitempty"`
	Cluster  *Cluster        `yaml:"cluster,omitempty"`
	Services *ServiceConfig  `yaml:"services,omitempty"`
}

//ImageConfig defines the image specification
type ImageConfig struct {
	Name      *string         `yaml:"name"`
	Tag       *string         `yaml:"tag"`
	BuildTime *string         `yaml:"buildTime"`
	Registry  *RegistryConfig `yaml:"registry"`
}

//DevSpaceConfig defines the devspace deployment
type DevSpaceConfig struct {
	Release        *Release                 `yaml:"release"`
	PortForwarding *[]*PortForwardingConfig `yaml:"portForwarding"`
	Sync           *[]*SyncConfig           `yaml:"sync"`
}

//ServiceConfig defines additional services
type ServiceConfig struct {
	Tiller           *TillerConfig     `yaml:"tiller,omitempty"`
	InternalRegistry *InternalRegistry `yaml:"internalRegistry,omitempty"`
}

//TillerConfig defines the tiller service
type TillerConfig struct {
	Release       *Release   `yaml:"release"`
	AppNamespaces *[]*string `yaml:"appNamespaces"`
}

//RegistryConfig defines the registry service
type RegistryConfig struct {
	URL      *string       `yaml:"url,omitempty"`
	Auth     *RegistryAuth `yaml:"auth,omitempty"`
	Insecure *bool         `yaml:"insecure,omitempty"`
}

//InternalRegistry defines the deployment of an internal registry
type InternalRegistry struct {
	Release *Release `yaml:"release,omitempty"`
	Host    *string  `yaml:"host,omitempty"`
}

//RegistryAuth is a user for the registry
type RegistryAuth struct {
	Username *string `yaml:"username"`
	Password *string `yaml:"password"`
}

//PortForwardingConfig defines the ports for a port forwarding to a DevSpace
type PortForwardingConfig struct {
	ResourceType  *string             `yaml:"resourceType"`
	LabelSelector *map[string]*string `yaml:"labelSelector"`
	PortMappings  *[]*PortMapping     `yaml:"portMappings"`
}

//PortMapping defines the ports for a PortMapping
type PortMapping struct {
	LocalPort  *int `yaml:"localPort"`
	RemotePort *int `yaml:"remotePort"`
}

//SyncConfig defines the paths for a SyncFolder
type SyncConfig struct {
	ResourceType         *string             `yaml:"resourceType"`
	LabelSelector        *map[string]*string `yaml:"labelSelector"`
	LocalSubPath         *string             `yaml:"localSubPath"`
	ContainerPath        *string             `yaml:"containerPath"`
	ExcludePaths         *[]string           `yaml:"excludePaths"`
	DownloadExcludePaths *[]string           `yaml:"downloadExcludePaths"`
	UploadExcludePaths   *[]string           `yaml:"uploadExcludePaths"`
}

//Release defines running version of a project
type Release struct {
	Name      *string                      `yaml:"name"`
	Namespace *string                      `yaml:"namespace"`
	Values    *map[interface{}]interface{} `yaml:"values,omitempty"`
}

//Cluster is a struct that contains data for a Kubernetes-Cluster
type Cluster struct {
	UseKubeConfig *bool   `yaml:"useKubeConfig,omitempty"`
	APIServer     *string `yaml:"apiServer,omitempty"`
	CaCert        *string `yaml:"caCert,omitempty"`
	User          *User   `yaml:"user,omitempty"`
}

//User is a user with its username and its client certificate
type User struct {
	Username   *string `yaml:"username,omitempty"`
	ClientCert *string `yaml:"clientCert,omitempty"`
	ClientKey  *string `yaml:"clientKey,omitempty"`
}
