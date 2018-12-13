package v1

// DeploymentConfig defines the configuration how the devspace should be deployed
type DeploymentConfig struct {
	Name       *string           `yaml:"name"`
	Namespace  *string           `yaml:"namespace,omitempty"`
	AutoReload *AutoReloadConfig `yaml:"autoReload,omitempty"`
	Helm       *HelmConfig       `yaml:"helm,omitempty"`
	Kubectl    *KubectlConfig    `yaml:"kubectl,omitempty"`
}

// HelmConfig defines the specific helm options used during deployment
type HelmConfig struct {
	ChartPath    *string `yaml:"chartPath,omitempty"`
	Wait         *bool   `yaml:"wait,omitempty"`
	Overwrite    *string `yaml:"overwrite,omitempty"`
	DevOverwrite *string `yaml:"devOverwrite,omitempty"`
}

// KubectlConfig defines the specific kubectl options used during deployment
type KubectlConfig struct {
	CmdPath   *string    `yaml:"cmdPath,omitempty"`
	Manifests *[]*string `yaml:"manifests,omitempty"`
}

// AutoReloadConfig defines the struct for auto reloading deployments and images
type AutoReloadConfig struct {
	Disabled *bool `yaml:"disabled,omitempty"`
}
