package latest

// Version is the version of the providers config
const Version = "v1beta1"

// Config holds all the different providers and their configuration
type Config struct {
	Version   string      `yaml:"version,omitempty"`
	Default   string      `yaml:"default,omitempty"`
	Providers []*Provider `yaml:"providers,omitempty"`
}

// Provider describes the struct to hold the cloud configuration
type Provider struct {
	Name string `yaml:"name,omitempty"`
	Host string `yaml:"host,omitempty"`

	// Key is used to obtain a token from the auth server
	Key string `yaml:"key,omitempty"`

	// Token is the actual authorization bearer
	Token string `yaml:"token,omitempty"`

	ClusterKey map[int]string `yaml:"clusterKeys,omitempty"`

	// These are the cached space tokens
	Spaces map[int]*SpaceCache `yaml:"spaces,omitempty"`
}

// SpaceCache holds the information for a specific space
type SpaceCache struct {
	Space          *Space          `yaml:"space"`
	ServiceAccount *ServiceAccount `yaml:"serviceAccount"`

	// The kube context
	KubeContext string `yaml:"kubeContext"`

	// Expires specifies when the token will expire
	LastResume int64 `yaml:"lastResume,omitempty"`
	Expires    int64 `yaml:"expires,omitempty"`
}

// Space holds the information about a space in the cloud
type Space struct {
	SpaceID      int            `yaml:"spaceID"`
	Name         string         `yaml:"name"`
	Namespace    string         `yaml:"namespace"`
	Owner        *Owner         `yaml:"account"`
	ProviderName string         `yaml:"providerName"`
	Cluster      *Cluster       `yaml:"cluster"`
	Created      string         `yaml:"created"`
	Domains      []*SpaceDomain `yaml:"domains"`
}

// SpaceDomain holds the information about a space domain
type SpaceDomain struct {
	DomainID int    `yaml:"id" json:"id"`
	URL      string `yaml:"url" json:"url"`
}

// ServiceAccount holds the information about a service account for a certain space
type ServiceAccount struct {
	SpaceID   int    `yaml:"spaceID"`
	Namespace string `yaml:"namespace"`
	CaCert    string `yaml:"caCert"`
	Server    string `yaml:"server"`
	Token     string `yaml:"token"`
}

// Project is the type that holds the project information
type Project struct {
	ProjectID int      `json:"id"`
	OwnerID   int      `json:"owner_id"`
	Cluster   *Cluster `json:"cluster"`
	Name      string   `json:"name"`
}

// Cluster is the type that holds the cluster information
type Cluster struct {
	ClusterID    int     `json:"id"`
	Server       *string `json:"server"`
	Owner        *Owner  `json:"account"`
	Name         string  `json:"name"`
	EncryptToken bool    `json:"encrypt_token"`
	CreatedAt    *string `json:"created_at"`
}

// Owner holds the information about a certain
type Owner struct {
	OwnerID int    `json:"id"`
	Name    string `json:"name"`
}

// ClusterUser is the type that golds the cluster user information
type ClusterUser struct {
	ClusterUserID int  `json:"id"`
	AccountID     int  `json:"account_id"`
	ClusterID     int  `json:"cluster_id"`
	IsAdmin       bool `json:"is_admin"`
}

// Registry is the type that holds the docker image registry information
type Registry struct {
	RegistryID int    `json:"id"`
	URL        string `json:"url"`
}
