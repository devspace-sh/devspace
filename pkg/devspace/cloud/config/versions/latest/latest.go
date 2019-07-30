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
}
