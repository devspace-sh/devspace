package v1

// Version is the current api version
const Version string = "v1"

//DevSpaceConfig defines the config for a DevSpace
type DevSpaceConfig struct {
	Version        string                      `yaml:"version"`
	PortForwarding []*PortForwarding           `yaml:"portForwarding"`
	SyncPaths      []*SyncPath                 `yaml:"syncPath"`
	Registry       map[interface{}]interface{} `yaml:"registry,omitempty"`
}

//PortForwarding defines the ports for a port forwarding to a DevSpace
type PortForwarding struct {
	ResourceType  string            `yaml:"resourceType"`
	LabelSelector map[string]string `yaml:"labelSelector"`
	PortMappings  []*PortMapping    `yaml:"portMappings"`
}

//PortMapping defines the ports for a PortMapping
type PortMapping struct {
	LocalPort  int `yaml:"localPort"`
	RemotePort int `yaml:"remotePort"`
}

//SyncPath defines the paths for a SyncFolder
type SyncPath struct {
	ResourceType  string            `yaml:"resourceType"`
	LabelSelector map[string]string `yaml:"labelSelector"`
	LocalSubPath  string            `yaml:"localSubPath"`
	ContainerPath string            `yaml:"containerPath"`
	ExcludeRegex  []string          `yaml:"excludeRegex"`
}

//PrivateConfig defines the private config of the users' computer
type PrivateConfig struct {
	Version  string          `yaml:"version"`
	Release  *Release        `yaml:"release"`
	Registry *RegistryAccess `yaml:"registry"`
	Cluster  *Cluster        `yaml:"cluster"`
}

//Release defines running version of a project
type Release struct {
	Name        string `yaml:"name"`
	Namespace   string `yaml:"namespace"`
	LatestBuild string `yaml:"latestBuild"`
	LatestImage string `yaml:"latestImage"`
}

//RegistryAccess sets the access from a user to a release
type RegistryAccess struct {
	Release *Release      `yaml:"release"`
	User    *RegistryUser `yaml:"user"`
}

//RegistryUser is a user for the registry
type RegistryUser struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

//Cluster is a struct that contains data for a Kubernetes-Cluster
type Cluster struct {
	TillerNamespace string `yaml:"tillerNamespace"`
	UseKubeConfig   bool   `yaml:"useKubeConfig,omitempty"`
	ApiServer       string `yaml:"apiServer,omitempty"`
	CaCert          string `yaml:"caCert,omitempty"`
	User            *User  `yaml:"user,omitempty"`
}

//User is a user with its username and its client certificate
type User struct {
	Username   string `yaml:"username,omitempty"`
	ClientCert string `yaml:"clientCert,omitempty"`
	ClientKey  string `yaml:"clientKey,omitempty"`
}

//AppConfig is the config for a single app
type AppConfig struct {
	Name      string
	Container *AppContainer
	External  *AppExternal
}

//AppContainer is the container in which an app is running
type AppContainer struct {
	Image string
	Ports []int
}

//AppExternal defines the external acces to an app
type AppExternal struct {
	Domain string
	Port   int
}
