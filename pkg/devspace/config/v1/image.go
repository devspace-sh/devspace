package v1

//ImageConfig defines the image specification
type ImageConfig struct {
	Name             *string           `yaml:"name"`
	Tag              *string           `yaml:"tag,omitempty"`
	Registry         *string           `yaml:"registry,omitempty"`
	CreatePullSecret *bool             `yaml:"createPullSecret,omitempty"`
	SkipPush         *bool             `yaml:"skipPush,omitempty"`
	AutoReload       *AutoReloadConfig `yaml:"autoReload,omitempty"`
	Build            *BuildConfig      `yaml:"build,omitempty"`
}

//BuildConfig defines the build process for an image
type BuildConfig struct {
	Disabled       *bool         `yaml:"disabled,omitempty"`
	ContextPath    *string       `yaml:"contextPath"`
	DockerfilePath *string       `yaml:"dockerfilePath"`
	Kaniko         *KanikoConfig `yaml:"kaniko,omitempty"`
	Docker         *DockerConfig `yaml:"docker,omitempty"`
	Options        *BuildOptions `yaml:"options,omitempty"`
}

// KanikoConfig tells the DevSpace CLI to build with Docker on Minikube or on localhost
type KanikoConfig struct {
	Cache      *bool   `yaml:"cache"`
	Namespace  *string `yaml:"namespace,omitempty"`
	PullSecret *string `yaml:"pullSecret,omitempty"`
}

// DockerConfig tells the DevSpace CLI to build with Docker on Minikube or on localhost
type DockerConfig struct {
	PreferMinikube *bool `yaml:"preferMinikube,omitempty"`
}

//BuildOptions defines options for building Docker images
type BuildOptions struct {
	BuildArgs *map[string]*string `yaml:"buildArgs,omitempty"`
	Target    *string             `yaml:"target,omitempty"`
	Network   *string             `yaml:"network,omitempty"`
}
