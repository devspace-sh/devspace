package v1

//RegistryConfig defines the registry service
type RegistryConfig struct {
	URL      *string       `yaml:"url,omitempty"`
	Auth     *RegistryAuth `yaml:"auth,omitempty"`
	Insecure *bool         `yaml:"insecure,omitempty"`
}

//RegistryAuth is a user for the registry
type RegistryAuth struct {
	Username *string `yaml:"username"`
	Password *string `yaml:"password"`
}
