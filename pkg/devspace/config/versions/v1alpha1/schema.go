package v1alpha1

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/util/ptr"
)

// Version is the current api version
const Version string = "v1alpha1"

// GetVersion returns the version
func (c *Config) GetVersion() string {
	return Version
}

// New creates a new config object
func New() config.Config {
	return &Config{
		Version: ptr.String(Version),
		Cluster: &Cluster{
			User: &ClusterUser{},
		},
		DevSpace: &DevSpaceConfig{
			Terminal: &Terminal{},
		},
		Images:     &map[string]*ImageConfig{},
		Registries: &map[string]*RegistryConfig{},
	}
}

// Config defines the configuration
type Config struct {
	Version          *string                     `yaml:"version"`
	DevSpace         *DevSpaceConfig             `yaml:"devSpace,omitempty"`
	Images           *map[string]*ImageConfig    `yaml:"images,omitempty"`
	Registries       *map[string]*RegistryConfig `yaml:"registries,omitempty"`
	Cluster          *Cluster                    `yaml:"cluster,omitempty"`
	Tiller           *TillerConfig               `yaml:"tiller,omitempty"`
	InternalRegistry *InternalRegistryConfig     `yaml:"internalRegistry,omitempty"`
}

// TillerConfig defines the tiller service
type TillerConfig struct {
	Namespace *string `yaml:"namespace,omitempty"`
}

// InternalRegistryConfig defines the internal registry config options
type InternalRegistryConfig struct {
	Deploy    *bool   `yaml:"deploy,omitempty"`
	Namespace *string `yaml:"namespace,omitempty"`
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
	Name       *string           `yaml:"name"`
	Namespace  *string           `yaml:"namespace,omitempty"`
	AutoReload *AutoReloadConfig `yaml:"autoReload,omitempty"`
	Helm       *HelmConfig       `yaml:"helm,omitempty"`
	Kubectl    *KubectlConfig    `yaml:"kubectl,omitempty"`
}

// HelmConfig defines the specific helm options used during deployment
type HelmConfig struct {
	ChartPath       *string                      `yaml:"chartPath,omitempty"`
	Wait            *bool                        `yaml:"wait,omitempty"`
	TillerNamespace *string                      `yaml:"tillerNamespace,omitempty"`
	DevOverwrite    *string                      `yaml:"devOverwrite,omitempty"`
	Override        *string                      `yaml:"override,omitempty"`
	OverrideValues  *map[interface{}]interface{} `yaml:"overrideValues,omitempty"`
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

//DevSpaceConfig defines the devspace deployment
type DevSpaceConfig struct {
	Terminal    *Terminal                `yaml:"terminal,omitempty"`
	AutoReload  *AutoReloadPathsConfig   `yaml:"autoReload,omitempty"`
	Services    *[]*ServiceConfig        `yaml:"services,omitempty"`
	Deployments *[]*DeploymentConfig     `yaml:"deployments,omitempty"`
	Ports       *[]*PortForwardingConfig `yaml:"ports,omitempty"`
	Sync        *[]*SyncConfig           `yaml:"sync,omitempty"`
}

// AutoReloadPathsConfig defines the struct for auto reloading devspace with additional paths
type AutoReloadPathsConfig struct {
	Paths *[]*string `yaml:"paths,omitempty"`
}

// ServiceConfig defines the kubernetes services that belong to the devspace
type ServiceConfig struct {
	Name          *string             `yaml:"name,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector"`
	ContainerName *string             `yaml:"containerName,omitempty"`
}

// PortForwardingConfig defines the ports for a port forwarding to a DevSpace
type PortForwardingConfig struct {
	Service       *string             `yaml:"service,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
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
	Service              *string             `yaml:"service,omitempty"`
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

// Terminal describes the terminal options
type Terminal struct {
	Disabled      *bool               `yaml:"disabled,omitempty"`
	Service       *string             `yaml:"service,omitempty"`
	ResourceType  *string             `yaml:"resourceType,omitempty"`
	LabelSelector *map[string]*string `yaml:"labelSelector,omitempty"`
	Namespace     *string             `yaml:"namespace,omitempty"`
	ContainerName *string             `yaml:"containerName,omitempty"`
	Command       *[]*string          `yaml:"command,omitempty"`
}

//RegistryConfig defines the registry service
type RegistryConfig struct {
	URL      *string       `yaml:"url,omitempty"`
	Auth     *RegistryAuth `yaml:"auth,omitempty"`
	Insecure *bool         `yaml:"insecure,omitempty"`
}

//RegistryAuth is a user for the registry
type RegistryAuth struct {
	Username *string `yaml:"username"`
	Password *string `yaml:"password"`
}
