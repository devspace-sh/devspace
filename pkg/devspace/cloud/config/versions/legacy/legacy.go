package legacy

// Config holds all the different providers and their configuration
type Config map[string]*Provider

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
