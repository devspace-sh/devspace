package v1

const Version string = "v1"

type DevSpaceConfig struct {
	Version        string                      `yaml:"version"`
	PortForwarding []*PortForwarding           `yaml:"portForwarding"`
	SyncPaths      []*SyncPath                 `yaml:"syncPath"`
	Registry       map[interface{}]interface{} `yaml:"registry,omitempty"`
}

type PortForwarding struct {
	ResourceType  string            `yaml:"resourceType"`
	LabelSelector map[string]string `yaml:"labelSelector"`
	PortMappings  []*PortMapping    `yaml:"portMappings"`
}

type PortMapping struct {
	LocalPort  int `yaml:"localPort"`
	RemotePort int `yaml:"remotePort"`
}

type SyncPath struct {
	ResourceType  string            `yaml:"resourceType"`
	LabelSelector map[string]string `yaml:"labelSelector"`
	LocalSubPath  string            `yaml:"localSubPath"`
	ContainerPath string            `yaml:"containerPath"`
	ExcludeRegex  []string          `yaml:"excludeRegex"`
}

type PrivateConfig struct {
	Version  string          `yaml:"version"`
	Release  *Release        `yaml:"release"`
	Registry *RegistryAccess `yaml:"registry"`
	Cluster  *Cluster        `yaml:"cluster"`
}

type Release struct {
	Name        string `yaml:"name"`
	Namespace   string `yaml:"namespace"`
	LatestBuild string `yaml:"latestBuild"`
	LatestImage string `yaml:"latestImage"`
}

type RegistryAccess struct {
	Release *Release      `yaml:"release"`
	User    *RegistryUser `yaml:"user"`
}

type RegistryUser struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Cluster struct {
	TillerNamespace string `yaml:"tillerNamespace"`
	UseKubeConfig   bool   `yaml:"useKubeConfig,omitempty"`
	ApiServer       string `yaml:"apiServer,omitempty"`
	CaCert          string `yaml:"caCert,omitempty"`
	User            *User  `yaml:"user,omitempty"`
}

type User struct {
	Username   string `yaml:"username,omitempty"`
	ClientCert string `yaml:"clientCert,omitempty"`
	ClientKey  string `yaml:"clientKey,omitempty"`
}

type AppConfig struct {
	Name      string
	Container *AppContainer
	External  *AppExternal
}

type AppContainer struct {
	Image string
	Port  int
}

type AppExternal struct {
	Domain string
	Port   int
}
