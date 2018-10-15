package v1

//ServiceConfig defines additional services
type ServiceConfig struct {
	Tiller           *TillerConfig     `yaml:"tiller,omitempty"`
	InternalRegistry *InternalRegistry `yaml:"internalRegistry,omitempty"`
}

//TillerConfig defines the tiller service
type TillerConfig struct {
	Release       *Release   `yaml:"release,omitempty"`
	AppNamespaces *[]*string `yaml:"appNamespaces,omitempty"`
}

//InternalRegistry defines the deployment of an internal registry
type InternalRegistry struct {
	Release *Release `yaml:"release,omitempty"`
}
