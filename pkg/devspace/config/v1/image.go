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
	ContextPath     *string      `yaml:"contextPath"`
	DockerfilePath  *string      `yaml:"dockerfilePath"`
	Engine          *BuildEngine `yaml:"engine"`
	LatestTimestamp *string      `yaml:"latestTime"`
}

//BuildEngine defines which build engine to use
type BuildEngine struct {
	Kaniko *KanikoBuildEngine `yaml:"kaniko"`
	Docker *DockerBuildEngine `yaml:"docker"`
}

//KanikoBuildEngine tells the DevSpace CLI to build with Docker on Minikube or on localhost
type KanikoBuildEngine struct {
	Enabled *bool `yaml:"enabled"`
}

//DockerBuildEngine tells the DevSpace CLI to build with Docker on Minikube or on localhost
type DockerBuildEngine struct {
	Enabled        *bool                     `yaml:"enabled"`
	PreferMinikube *bool                     `yaml:"preferMinikube"`
	Options        *DockerBuildEngineOptions `yaml:"options"`
}

//DockerBuildEngineOptions defines options for building with DockerBuildEngine
type DockerBuildEngineOptions struct {
	BuildArgs *map[string]*string `yaml:"buildArgs"`
}
