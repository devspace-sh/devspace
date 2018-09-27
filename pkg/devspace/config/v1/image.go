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
	ContextPath     *string       `yaml:"contextPath"`
	DockerfilePath  *string       `yaml:"dockerfilePath"`
	Engine          *BuildEngine  `yaml:"engine"`
	LatestTimestamp *string       `yaml:"latestTimestamp"`
	Options         *BuildOptions `yaml:"options"`
}

//BuildEngine defines which build engine to use
type BuildEngine struct {
	Kaniko *KanikoBuildEngine `yaml:"kaniko"`
	Docker *DockerBuildEngine `yaml:"docker"`
}

//KanikoBuildEngine tells the DevSpace CLI to build with Docker on Minikube or on localhost
type KanikoBuildEngine struct {
	Enabled   *bool   `yaml:"enabled"`
	Namespace *string `yaml:"namespace"`
}

//DockerBuildEngine tells the DevSpace CLI to build with Docker on Minikube or on localhost
type DockerBuildEngine struct {
	Enabled        *bool `yaml:"enabled"`
	PreferMinikube *bool `yaml:"preferMinikube"`
}

//BuildOptions defines options for building Docker images
type BuildOptions struct {
	BuildArgs *map[string]*string `yaml:"buildArgs"`
	Target    *string             `yaml:"target"`
	Network   *string             `yaml:"network"`
}
