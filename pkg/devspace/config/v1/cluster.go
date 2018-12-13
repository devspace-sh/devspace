package v1

//Cluster is a struct that contains data for a Kubernetes-Cluster
type Cluster struct {
	CloudProvider             *string      `yaml:"cloudProvider,omitempty"`
	CloudTarget               *string      `yaml:"cloudTarget,omitempty"`
	CloudProviderDeployTarget *string      `yaml:"cloudProviderDeployTarget,omitempty"` // DEPRECATED
	KubeContext               *string      `yaml:"kubeContext,omitempty"`
	Namespace                 *string      `yaml:"namespace,omitempty"`
	APIServer                 *string      `yaml:"apiServer,omitempty"`
	CaCert                    *string      `yaml:"caCert,omitempty"`
	User                      *ClusterUser `yaml:"user,omitempty"`
}

//ClusterUser is a user with its username and its client certificate
type ClusterUser struct {
	ClientCert *string `yaml:"clientCert,omitempty"`
	ClientKey  *string `yaml:"clientKey,omitempty"`
	Token      *string `yaml:"token,omitempty"`
}
