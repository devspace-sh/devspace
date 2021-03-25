package generated

import "gopkg.in/yaml.v2"

// Config specifies the runtime config struct
type Config struct {
	OverrideProfile *string                 `yaml:"lastOverrideProfile,omitempty"`
	ActiveProfile   string                  `yaml:"activeProfile,omitempty"`
	Vars            map[string]string       `yaml:"vars,omitempty"`
	VarsEncrypted   bool                    `yaml:"varsEncrypted,omitempty"`
	Profiles        map[string]*CacheConfig `yaml:"profiles,omitempty"`
}

// DeepCopy creates a deep copy of the config
func (c *Config) DeepCopy() *Config {
	o, _ := yaml.Marshal(c)
	n := &Config{}
	_ = yaml.Unmarshal(o, n)
	return n
}

// LastContextConfig holds all the informations about the last used kubernetes context
type LastContextConfig struct {
	Namespace string `yaml:"namespace,omitempty"`
	Context   string `yaml:"context,omitempty"`
}

// CacheConfig holds all the information specific to a certain config
type CacheConfig struct {
	Deployments  map[string]*DeploymentCache `yaml:"deployments,omitempty"`
	Images       map[string]*ImageCache      `yaml:"images,omitempty"`
	Dependencies map[string]string           `yaml:"dependencies,omitempty"`
	LastContext  *LastContextConfig          `yaml:"lastContext,omitempty"`
}

// ImageCache holds the cache related information about a certain image
type ImageCache struct {
	ImageConfigHash string `yaml:"imageConfigHash,omitempty"`

	DockerfileHash string `yaml:"dockerfileHash,omitempty"`
	ContextHash    string `yaml:"contextHash,omitempty"`
	EntrypointHash string `yaml:"entrypointHash,omitempty"`

	CustomFilesHash string `yaml:"customFilesHash,omitempty"`

	ImageName string `yaml:"imageName,omitempty"`
	Tag       string `yaml:"tag,omitempty"`
}

// DeploymentCache holds the information about a specific deployment
type DeploymentCache struct {
	DeploymentConfigHash string `yaml:"deploymentConfigHash,omitempty"`

	HelmOverridesHash   string `yaml:"helmOverridesHash,omitempty"`
	HelmChartHash       string `yaml:"helmChartHash,omitempty"`
	HelmReleaseRevision string `yaml:"helmReleaseRevision,omitempty"`

	KubectlManifestsHash string `yaml:"kubectlManifestsHash,omitempty"`
}
