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
	SpaceToken map[int]*SpaceToken `yaml:"spaceTokens,omitempty"`
}

// SpaceToken holds the information for a specific space
type SpaceToken struct {
	// Service account information
	Token     string `yaml:"token"`
	Namespace string `yaml:"namespace"`
	Server    string `yaml:"server"`
	CaCert    string `yaml:"caCert"`

	// Cluster information
	ClusterID           int    `yaml:"clusterID"`
	ClusterName         string `yaml:"clusterName"`
	ClusterEncryptToken bool   `yaml:"clusterEncryptToken"`

	// Expires specifies when the token will expire
	LastResume int64 `yaml:"lastResume,omitempty"`
	Expires    int64 `yaml:"expires,omitempty"`
}
