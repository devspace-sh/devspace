package v1

// DeploymentConfig defines the configuration how the devspace should be deployed
type DeploymentConfig struct {
	Name      *string        `yaml:"name"`
	Namespace *string        `yaml:"namespace,omitempty"`
	Helm      *HelmConfig    `yaml:"helm,omitempty"`
	Kubectl   *KubectlConfig `yaml:"kubectl,omitempty"`
}

// HelmConfig defines the specific helm options used during deployment
type HelmConfig struct {
	ChartPath    *string `yaml:"chartPath,omitempty"`
	DevOverwrite *string `yaml:"devOverwrite,omitempty"`
}

// KubectlConfig defines the specific kubectl options used during deployment
type KubectlConfig struct {
	CmdPath   *string    `yaml:"cmdPath,omitempty"`
	Manifests *[]*string `yaml:"manifests,omitempty"`
}
