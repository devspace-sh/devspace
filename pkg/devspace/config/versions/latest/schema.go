package latest

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	v1 "k8s.io/api/core/v1"
)

// Version is the current api version
const Version string = "v1alpha4"

// GetVersion returns the version
func (c *Config) GetVersion() string {
	return Version
}

// New creates a new config object
func New() config.Config {
	return &Config{
		Version: ptr.String(Version),
		Cluster: &Cluster{},
		Dev:     &DevConfig{},
		Images:  &map[string]*ImageConfig{},
	}
}

// Config defines the configuration
type Config struct {
	Version     *string                  `yaml:"version"`
	Images      *map[string]*ImageConfig `yaml:"images,omitempty"`
	Deployments *[]*DeploymentConfig     `yaml:"deployments,omitempty"`
	Dev         *DevConfig               `yaml:"dev,omitempty"`
	Cluster     *Cluster                 `yaml:"cluster,omitempty"`
}

// Cluster is a struct that contains data for a Kubernetes-Cluster
type Cluster struct {
	KubeContext *string      `yaml:"kubeContext,omitempty"`
	Namespace   *string      `yaml:"namespace,omitempty"`
	APIServer   *string      `yaml:"apiServer,omitempty"`
	CaCert      *string      `yaml:"caCert,omitempty"`
	User        *ClusterUser `yaml:"user,omitempty"`
}

// ClusterUser is a user with its username and its client certificate
type ClusterUser struct {
	ClientCert *string `yaml:"clientCert,omitempty"`
	ClientKey  *string `yaml:"clientKey,omitempty"`
	Token      *string `yaml:"token,omitempty"`
}

// DeploymentConfig defines the configuration how the devspace should be deployed
type DeploymentConfig struct {
	Name      *string          `yaml:"name"`
	Namespace *string          `yaml:"namespace,omitempty"`
	Helm      *HelmConfig      `yaml:"helm,omitempty"`
	Kubectl   *KubectlConfig   `yaml:"kubectl,omitempty"`
	Component *ComponentConfig `yaml:"component,omitempty"`
}

// ComponentConfig holds the component information
type ComponentConfig struct {
	Containers          *[]*ContainerConfig  `yaml:"containers,omitempty"`
	Replicas            *int                 `yaml:"replicas,omitempty"`
	PodManagementPolicy *string              `yaml:"podManagementPolicy,omitempty"`
	ServiceName         *string              `yaml:"serviceName,omitempty"`
	RollingUpdate       *RollingUpdateConfig `yaml:"rollingUpdate,omitempty"`
	Volumes             *[]*VolumeConfig     `yaml:"volumes,omitempty"`
	Service             *ServiceConfig       `yaml:"service,omitempty"`
	PullSecrets         *[]*string           `yaml:"pullSecrets,omitempty"`
	Autoscaling         *AutoScalingConfig   `yaml:"autoScaling,omitempty"`
}

// AutoScalingConfig holds the autoscaling config of a component
type AutoScalingConfig struct {
	Horizontal *AutoScalingHorizontalConfig `yaml:"horizontal,omitempty"`
}

// AutoScalingHorizontalConfig holds the horizontal autoscaling config of a component
type AutoScalingHorizontalConfig struct {
	MaxReplicas   *int    `yaml:"maxReplicas,omitempty"`
	AverageCPU    *string `yaml:"averageCPU,omitempty"`
	AverageMemory *string `yaml:"averageMemory,omitempty"`
}

// RollingUpdateConfig holds the configuration for rolling updates
type RollingUpdateConfig struct {
	Enabled        *bool   `yaml:"enabled,omitempty"`
	MaxSurge       *string `yaml:"maxSurge,omitempty"`
	MaxUnavailable *string `yaml:"maxUnavailable,omitempty"`
	Partition      *int    `yaml:"partition,omitempty"`
}

// ContainerConfig holds the configurations of a container
type ContainerConfig struct {
	Name           *string                         `yaml:"name,omitempty"`
	Image          *string                         `yaml:"image,omitempty"`
	Resources      *map[interface{}]interface{}    `yaml:"resources,omitempty"`
	Env            *[]*map[interface{}]interface{} `yaml:"env,omitempty"`
	VolumeMounts   *[]*VolumeMountConfig           `yaml:"volumeMounts,omitempty"`
	LifenessProbe  *map[interface{}]interface{}    `yaml:"lifenessProbe,omitempty"`
	ReadinessProbe *map[interface{}]interface{}    `yaml:"readinessProbe,omitempty"`
	Command        *[]*string                      `yaml:"command,omitempty"`
	Args           *[]*string                      `yaml:"args,omitempty"`
}

// VolumeMountConfig holds the configuration for a specific mount path
type VolumeMountConfig struct {
	ContainerPath *string                  `yaml:"containerPath,omitempty"`
	Volume        *VolumeMountVolumeConfig `yaml:"volume,omitempty"`
}

// VolumeMountVolumeConfig holds the configuration for a specfic mount path volume
type VolumeMountVolumeConfig struct {
	Name     *string `yaml:"name,omitempty"`
	SubPath  *string `yaml:"subPath,omitempty"`
	ReadOnly *bool   `yaml:"readOnly,omitempty"`
}

// VolumeConfig holds the configuration for a specific volume
type VolumeConfig struct {
	Name      *string                   `yaml:"name,omitempty"`
	Size      *string                   `yaml:"size,omitempty"`
	ConfigMap *v1.ConfigMapVolumeSource `yaml:"configMap,omitempty"`
	Secret    *v1.SecretVolumeSource    `yaml:"secret,omitempty"`
}

// ServiceConfig holds the configuration of a component service
type ServiceConfig struct {
	Name  *string               `yaml:"name,omitempty"`
	Type  *string               `yaml:"type,omitempty"`
	Ports *[]*ServicePortConfig `yaml:"ports,omitempty"`
}

// ServicePortConfig holds the port configuration of a component service
type ServicePortConfig struct {
	Port          *int    `yaml:"port,omitempty"`
	ContainerPort *int    `yaml:"containerPort,omitempty"`
	Protocol      *string `yaml:"protocol,omitempty"`
}

// HelmConfig defines the specific helm options used during deployment
type HelmConfig struct {
	Chart           *ChartConfig                 `yaml:"chart,omitempty"`
	Wait            *bool                        `yaml:"wait,omitempty"`
	Rollback        *bool                        `yaml:"rollback,omitempty"`
	Force           *bool                        `yaml:"force,omitempty"`
	Timeout         *int64                       `yaml:"timeout,omitempty"`
	TillerNamespace *string                      `yaml:"tillerNamespace,omitempty"`
	DevSpaceValues  *bool                        `yaml:"devSpaceValues,omitempty"`
	ValuesFiles     *[]*string                   `yaml:"valuesFiles,omitempty"`
	Values          *map[interface{}]interface{} `yaml:"values,omitempty"`
}

// ChartConfig defines the helm chart options
type ChartConfig struct {
	Name     *string `yaml:"name,omitempty"`
	RepoURL  *string `yaml:"repo,omitempty"`
	Username *string `yaml:"username,omitempty"`
	Password *string `yaml:"password,omitempty"`
	Version  *string `yaml:"version,omitempty"`
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
	Entrypoint *[]*string `yaml:"entrypoint,omitempty"`
	Dockerfile *string    `yaml:"dockerfile,omitempty"`
	Context    *string    `yaml:"context,omitempty"`
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

// BuildConfig defines the build process for an image
type BuildConfig struct {
	Disabled   *bool         `yaml:"disabled,omitempty"`
	Context    *string       `yaml:"context,omitempty"`
	Dockerfile *string       `yaml:"dockerfile,omitempty"`
	Kaniko     *KanikoConfig `yaml:"kaniko,omitempty"`
	Docker     *DockerConfig `yaml:"docker,omitempty"`
	Options    *BuildOptions `yaml:"options,omitempty"`
}

// KanikoConfig tells the DevSpace CLI to build with Docker on Minikube or on localhost
type KanikoConfig struct {
	Cache        *bool   `yaml:"cache"`
	SnapshotMode *string `yaml:"snapshotMode,omitempty"`
	Namespace    *string `yaml:"namespace,omitempty"`
	PullSecret   *string `yaml:"pullSecret,omitempty"`
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
