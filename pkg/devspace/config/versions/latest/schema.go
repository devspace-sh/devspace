package latest

import "github.com/covexo/devspace/pkg/devspace/config/versions/config"

// Version is the current api version
const Version string = "v1alpha2"

// GetVersion returns the version
func (c *Config) GetVersion() string {
	return Version
}

// New creates a new config object
func New() config.Config {
	return &Config{
		Cluster: &Cluster{},
		Dev:     &DevConfig{},
		Images:  &map[string]*ImageConfig{},
	}
}

// Config defines the configuration
type Config struct {
	Version     *string                  `yaml:"version"`
	Cluster     *Cluster                 `yaml:"cluster,omitempty"`
	Dev         *DevConfig               `yaml:"dev,omitempty"`
	Deployments *[]*DeploymentConfig     `yaml:"deployments,omitempty"`
	Images      *map[string]*ImageConfig `yaml:"images,omitempty"`
}

// Cluster is a struct that contains data for a Kubernetes-Cluster
type Cluster struct {
	CloudProvider *string      `yaml:"cloudProvider,omitempty"`
	KubeContext   *string      `yaml:"kubeContext,omitempty"`
	Namespace     *string      `yaml:"namespace,omitempty"`
	APIServer     *string      `yaml:"apiServer,omitempty"`
	CaCert        *string      `yaml:"caCert,omitempty"`
	User          *ClusterUser `yaml:"user,omitempty"`
}

// ClusterUser is a user with its username and its client certificate
type ClusterUser struct {
	ClientCert *string `yaml:"clientCert,omitempty"`
	ClientKey  *string `yaml:"clientKey,omitempty"`
	Token      *string `yaml:"token,omitempty"`
}

// DeploymentConfig defines the configuration how the devspace should be deployed
type DeploymentConfig struct {
	Name      *string        `yaml:"name"`
	Namespace *string        `yaml:"namespace,omitempty"`
	Helm      *HelmConfig    `yaml:"helm,omitempty"`
	Kubectl   *KubectlConfig `yaml:"kubectl,omitempty"`
}

// HelmConfig defines the specific helm options used during deployment
type HelmConfig struct {
	ChartPath       *string                      `yaml:"chartPath,omitempty"`
	Wait            *bool                        `yaml:"wait,omitempty"`
	TillerNamespace *string                      `yaml:"tillerNamespace,omitempty"`
	Overrides       *[]*string                   `yaml:"overrides,omitempty"`
	OverrideValues  *map[interface{}]interface{} `yaml:"overrideValues,omitempty"`
}

// KubectlConfig defines the specific kubectl options used during deployment
type KubectlConfig struct {
	CmdPath   *string    `yaml:"cmdPath,omitempty"`
	Manifests *[]*string `yaml:"manifests,omitempty"`
}

// DevConfig defines the devspace deployment
type DevConfig struct {
	Terminal       *Terminal                `yaml:"terminal,omitempty"`
	AutoReload     *AutoReloadConfig        `yaml:"autoReload,omitempty"`
	OverrideImages *[]*ImageOverrideConfig  `yaml:"overrideImages,omitempty"`
	Selectors      *[]*SelectorConfig       `yaml:"selectors,omitempty"`
	Ports          *[]*PortForwardingConfig `yaml:"ports,omitempty"`
	Sync           *[]*SyncConfig           `yaml:"sync,omitempty"`
}

// ImageOverrideConfig holds information about what parts of the image config are overwritten during devspace dev
type ImageOverrideConfig struct {
	Name       *string    `yaml:"name"`
	Entrypoint *[]*string `yaml:"entrypoint"`
}

// AutoReloadConfig defines the struct for auto reloading devspace with additional paths
type AutoReloadConfig struct {
	Paths       *[]*string `yaml:"paths,omitempty"`
	Deployments *[]*string `yaml:"deployments,omitempty"`
	Images      *[]*string `yaml:"images,omitempty"`
}

// SelectorConfig defines the selectors that belong to the devspace
type SelectorConfig struct {
	Name          *string             `yaml:"name,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector"`
	ContainerName *string             `yaml:"containerName,omitempty"`
}

// PortForwardingConfig defines the ports for a port forwarding to a DevSpace
type PortForwardingConfig struct {
	Selector      *string             `yaml:"selector,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector,omitempty"`
	PortMappings  *[]*PortMapping     `yaml:"portMappings"`
}

// PortMapping defines the ports for a PortMapping
type PortMapping struct {
	LocalPort   *int    `yaml:"localPort"`
	RemotePort  *int    `yaml:"remotePort"`
	BindAddress *string `yaml:"bindAddress,omitempty"`
}

// SyncConfig defines the paths for a SyncFolder
type SyncConfig struct {
	Selector             *string             `yaml:"selector,omitempty"`
	Namespace            *string             `yaml:"namespace,omitempty"`
	LabelSelector        *map[string]*string `yaml:"labelSelector,omitempty"`
	ContainerName        *string             `yaml:"containerName,omitempty"`
	LocalSubPath         *string             `yaml:"localSubPath,omitempty"`
	ContainerPath        *string             `yaml:"containerPath,omitempty"`
	ExcludePaths         *[]string           `yaml:"excludePaths,omitempty"`
	DownloadExcludePaths *[]string           `yaml:"downloadExcludePaths,omitempty"`
	UploadExcludePaths   *[]string           `yaml:"uploadExcludePaths,omitempty"`
	BandwidthLimits      *BandwidthLimits    `yaml:"bandwidthLimits,omitempty"`
}

// BandwidthLimits defines the struct for specifying the sync bandwidth limits
type BandwidthLimits struct {
	Download *int64 `yaml:"download,omitempty"`
	Upload   *int64 `yaml:"upload,omitempty"`
}

// ImageConfig defines the image specification
type ImageConfig struct {
	Image            *string      `yaml:"image"`
	Tag              *string      `yaml:"tag,omitempty"`
	CreatePullSecret *bool        `yaml:"createPullSecret,omitempty"`
	Insecure         *bool        `yaml:"insecure,omitempty"`
	SkipPush         *bool        `yaml:"skipPush,omitempty"`
	Build            *BuildConfig `yaml:"build,omitempty"`
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

// BuildOptions defines options for building Docker images
type BuildOptions struct {
	BuildArgs *map[string]*string `yaml:"buildArgs,omitempty"`
	Target    *string             `yaml:"target,omitempty"`
	Network   *string             `yaml:"network,omitempty"`
}

// Terminal describes the terminal options
type Terminal struct {
	Disabled      *bool               `yaml:"disabled,omitempty"`
	Selector      *string             `yaml:"selector,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ContainerName *string             `yaml:"containerName,omitempty"`
	Command       *[]*string          `yaml:"command,omitempty"`
}
