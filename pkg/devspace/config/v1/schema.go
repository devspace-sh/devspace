package v1

// Version is the current api version
const Version string = "v1"

//DevSpaceConfig defines the config for a DevSpace
type Config struct {
	Version  *string         `yaml:"version"`
	DevSpace *DevSpaceConfig `yaml:"devSpace,omitempty"`
	Image    *ImageConfig    `yaml:"image,omitempty"`
	Cluster  *Cluster        `yaml:"cluster,omitempty"`
	Services *ServiceConfig  `yaml:"services,omitempty"`
}

type ServiceConfig struct {
	Tiller   *TillerConfig   `yaml:"tiller,omitempty"`
	Registry *RegistryConfig `yaml:"registry,omitempty"`
}

type ImageConfig struct {
	Name      *string `yaml:"name"`
	Tag       *string `yaml:"tag"`
	BuildTime *string `yaml:"buildTime"`
}

type TillerConfig struct {
	Release       *Release  `yaml:"release"`
	AppNamespaces []*string `yaml:"appNamespaces"`
}

type DevSpaceConfig struct {
	Release        *Release                `yaml:"release"`
	PortForwarding []*PortForwardingConfig `yaml:"portForwarding"`
	Sync           []*SyncConfig           `yaml:"sync"`
}

type RegistryConfig struct {
	External *string           `yaml:"external,omitempty"`
	Internal *InternalRegistry `yaml:"internal,omitempty"`
	User     *RegistryUser     `yaml:"user,omitempty"`
	Insecure *bool             `yaml:"insecure,omitempty"`
}

type InternalRegistry struct {
	Release *Release `yaml:"release,omitempty"`
	Host    *string  `yaml:"host,omitempty"`
}

//RegistryUser is a user for the registry
type RegistryUser struct {
	Username *string `yaml:"username"`
	Password *string `yaml:"password"`
}

//PortForwarding defines the ports for a port forwarding to a DevSpace
type PortForwardingConfig struct {
	ResourceType  *string            `yaml:"resourceType"`
	LabelSelector map[string]*string `yaml:"labelSelector"`
	PortMappings  []*PortMapping     `yaml:"portMappings"`
}

//PortMapping defines the ports for a PortMapping
type PortMapping struct {
	LocalPort  *int `yaml:"localPort"`
	RemotePort *int `yaml:"remotePort"`
}

//SyncPath defines the paths for a SyncFolder
type SyncConfig struct {
	ResourceType  *string            `yaml:"resourceType"`
	LabelSelector map[string]*string `yaml:"labelSelector"`
	LocalSubPath  *string            `yaml:"localSubPath"`
	ContainerPath *string            `yaml:"containerPath"`
	ExcludeRegex  []*string          `yaml:"excludeRegex"`
}

//PrivateConfig defines the private config of the users' computer
type PrivateConfig struct {
	Version  *string         `yaml:"version"`
	Release  *Release        `yaml:"release"`
	Tiller   *TillerConfig   `yaml:"tiller,omitempty"`
	Registry *RegistryConfig `yaml:"registry"`
}

//Release defines running version of a project
type Release struct {
	Name      *string                     `yaml:"name"`
	Namespace *string                     `yaml:"namespace"`
	Values    map[interface{}]interface{} `yaml:"internal,omitempty"`
}

//Cluster is a struct that contains data for a Kubernetes-Cluster
type Cluster struct {
	UseKubeConfig *bool   `yaml:"useKubeConfig,omitempty"`
	ApiServer     *string `yaml:"apiServer,omitempty"`
	CaCert        *string `yaml:"caCert,omitempty"`
	User          *User   `yaml:"user,omitempty"`
}

//User is a user with its username and its client certificate
type User struct {
	Username   *string `yaml:"username,omitempty"`
	ClientCert *string `yaml:"clientCert,omitempty"`
	ClientKey  *string `yaml:"clientKey,omitempty"`
}
