package v1

//ImageConfig defines the image specification
type ImageConfig struct {
	Name     *string      `yaml:"name"`
	Tag      *string      `yaml:"tag"`
	Registry *string      `yaml:"registry"`
	Build    *BuildConfig `yaml:"build"`
}

//BuildConfig defines the build process for an image
type BuildConfig struct {
	ContextPath    *string       `yaml:"contextPath"`
	DockerfilePath *string       `yaml:"dockerfilePath"`
	Kaniko         *KanikoConfig `yaml:"kaniko,omitempty"`
	Docker         *DockerConfig `yaml:"docker,omitempty"`
	Options        *BuildOptions `yaml:"options"`
}

// KanikoConfig tells the DevSpace CLI to build with Docker on Minikube or on localhost
type KanikoConfig struct {
	Cached    *bool   `yaml:"cached"`
	Namespace *string `yaml:"namespace"`
}

// DockerConfig tells the DevSpace CLI to build with Docker on Minikube or on localhost
type DockerConfig struct {
	PreferMinikube *bool `yaml:"preferMinikube"`
}

//BuildOptions defines options for building Docker images
type BuildOptions struct {
	BuildArgs *map[string]*string `yaml:"buildArgs"`
	Target    *string             `yaml:"target"`
	Network   *string             `yaml:"network"`
}
