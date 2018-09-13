package v1

//Cluster is a struct that contains data for a Kubernetes-Cluster
type Cluster struct {
	UseKubeConfig *bool        `yaml:"useKubeConfig,omitempty"`
	APIServer     *string      `yaml:"apiServer,omitempty"`
	CaCert        *string      `yaml:"caCert,omitempty"`
	User          *ClusterUser `yaml:"user,omitempty"`
}

//ClusterUser is a user with its username and its client certificate
type ClusterUser struct {
	Username   *string `yaml:"username,omitempty"`
	ClientCert *string `yaml:"clientCert,omitempty"`
	ClientKey  *string `yaml:"clientKey,omitempty"`
}
