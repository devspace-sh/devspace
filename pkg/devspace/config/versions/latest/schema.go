package latest

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
)

// Version is the current api version
const Version string = "v1beta5"

// GetVersion returns the version
func (c *Config) GetVersion() string {
	return Version
}

// New creates a new config object
func New() config.Config {
	return NewRaw()
}

// NewRaw creates a new config object
func NewRaw() *Config {
	return &Config{
		Version: Version,
		Dev:     &DevConfig{},
		Images:  map[string]*ImageConfig{},
	}
}

// Config defines the configuration
type Config struct {
	Version string `yaml:"version"`

	Images       map[string]*ImageConfig `yaml:"images,omitempty"`
	Deployments  []*DeploymentConfig     `yaml:"deployments,omitempty"`
	Dev          *DevConfig              `yaml:"dev,omitempty"`
	Dependencies []*DependencyConfig     `yaml:"dependencies,omitempty"`
	Hooks        []*HookConfig           `yaml:"hooks,omitempty"`
	Commands     []*CommandConfig        `yaml:"commands,omitempty"`

	Vars     []*Variable      `yaml:"vars,omitempty"`
	Profiles []*ProfileConfig `yaml:"profiles,omitempty"`
}

// ImageConfig defines the image specification
type ImageConfig struct {
	Image            string       `yaml:"image"`
	Tag              string       `yaml:"tag,omitempty"`
	Dockerfile       string       `yaml:"dockerfile,omitempty"`
	Context          string       `yaml:"context,omitempty"`
	Entrypoint       []string     `yaml:"entrypoint,omitempty"`
	Cmd              []string     `yaml:"cmd,omitempty"`
	CreatePullSecret *bool        `yaml:"createPullSecret,omitempty"`
	Build            *BuildConfig `yaml:"build,omitempty"`
}

// BuildConfig defines the build process for an image
type BuildConfig struct {
	Docker   *DockerConfig `yaml:"docker,omitempty"`
	Kaniko   *KanikoConfig `yaml:"kaniko,omitempty"`
	Custom   *CustomConfig `yaml:"custom,omitempty"`
	Disabled *bool         `yaml:"disabled,omitempty"`
}

// DockerConfig tells the DevSpace CLI to build with Docker on Minikube or on localhost
type DockerConfig struct {
	UseBuildKit     *bool         `yaml:"useBuildKit,omitempty"`
	PreferMinikube  *bool         `yaml:"preferMinikube,omitempty"`
	SkipPush        *bool         `yaml:"skipPush,omitempty"`
	DisableFallback *bool         `yaml:"disableFallback,omitempty"`
	Options         *BuildOptions `yaml:"options,omitempty"`
}

// KanikoConfig tells the DevSpace CLI to build with Docker on Minikube or on localhost
type KanikoConfig struct {
	Cache        *bool         `yaml:"cache,omitempty"`
	SnapshotMode string        `yaml:"snapshotMode,omitempty"`
	Flags        []string      `yaml:"flags,omitempty"`
	Namespace    string        `yaml:"namespace,omitempty"`
	Insecure     *bool         `yaml:"insecure,omitempty"`
	PullSecret   string        `yaml:"pullSecret,omitempty"`
	Options      *BuildOptions `yaml:"options,omitempty"`
}

// CustomConfig tells the DevSpace CLI to build with a custom build script
type CustomConfig struct {
	Command   string    `yaml:"command,omitempty"`
	Args      []*string `yaml:"flags,omitempty"`
	ImageFlag string    `yaml:"imageFlag,omitempty"`
	OnChange  []*string `yaml:"onChange,omitempty"`
}

// BuildOptions defines options for building Docker images
type BuildOptions struct {
	Target    string             `yaml:"target,omitempty"`
	Network   string             `yaml:"network,omitempty"`
	BuildArgs map[string]*string `yaml:"buildArgs,omitempty"`
}

// DeploymentConfig defines the configuration how the devspace should be deployed
type DeploymentConfig struct {
	Name      string         `yaml:"name"`
	Namespace string         `yaml:"namespace,omitempty"`
	Helm      *HelmConfig    `yaml:"helm,omitempty"`
	Kubectl   *KubectlConfig `yaml:"kubectl,omitempty"`
}

// ComponentConfig holds the component information
type ComponentConfig struct {
	InitContainers      []*ContainerConfig   `yaml:"initContainers,omitempty"`
	Containers          []*ContainerConfig   `yaml:"containers,omitempty"`
	Labels              map[string]string    `yaml:"labels,omitempty"`
	Annotations         map[string]string    `yaml:"annotations,omitempty"`
	Volumes             []*VolumeConfig      `yaml:"volumes,omitempty"`
	Service             *ServiceConfig       `yaml:"service,omitempty"`
	ServiceName         string               `yaml:"serviceName,omitempty"`
	Ingress             *IngressConfig       `yaml:"ingress,omitempty"`
	Replicas            *int                 `yaml:"replicas,omitempty"`
	Autoscaling         *AutoScalingConfig   `yaml:"autoScaling,omitempty"`
	RollingUpdate       *RollingUpdateConfig `yaml:"rollingUpdate,omitempty"`
	PullSecrets         []*string            `yaml:"pullSecrets,omitempty"`
	PodManagementPolicy string               `yaml:"podManagementPolicy,omitempty"`
}

// ContainerConfig holds the configurations of a container
type ContainerConfig struct {
	Name           string                        `yaml:"name,omitempty"`
	Image          string                        `yaml:"image,omitempty"`
	Command        []string                      `yaml:"command,omitempty"`
	Args           []string                      `yaml:"args,omitempty"`
	Stdin          bool                          `yaml:"stdin,omitempty"`
	TTY            bool                          `yaml:"tty,omitempty"`
	Env            []map[interface{}]interface{} `yaml:"env,omitempty"`
	VolumeMounts   []*VolumeMountConfig          `yaml:"volumeMounts,omitempty"`
	Resources      map[interface{}]interface{}   `yaml:"resources,omitempty"`
	LivenessProbe  map[interface{}]interface{}   `yaml:"livenessProbe,omitempty"`
	ReadinessProbe map[interface{}]interface{}   `yaml:"readinessProbe,omitempty"`
}

// VolumeMountConfig holds the configuration for a specific mount path
type VolumeMountConfig struct {
	ContainerPath string                   `yaml:"containerPath,omitempty"`
	Volume        *VolumeMountVolumeConfig `yaml:"volume,omitempty"`
}

// VolumeMountVolumeConfig holds the configuration for a specfic mount path volume
type VolumeMountVolumeConfig struct {
	Name     string `yaml:"name,omitempty"`
	SubPath  string `yaml:"subPath,omitempty"`
	ReadOnly *bool  `yaml:"readOnly,omitempty"`
}

// VolumeConfig holds the configuration for a specific volume
type VolumeConfig struct {
	Name        string                      `yaml:"name,omitempty"`
	Labels      map[string]string           `yaml:"labels,omitempty"`
	Annotations map[string]string           `yaml:"annotations,omitempty"`
	Size        string                      `yaml:"size,omitempty"`
	ConfigMap   map[interface{}]interface{} `yaml:"configMap,omitempty"`
	Secret      map[interface{}]interface{} `yaml:"secret,omitempty"`
}

// ServiceConfig holds the configuration of a component service
type ServiceConfig struct {
	Name        string               `yaml:"name,omitempty"`
	Labels      map[string]string    `yaml:"labels,omitempty"`
	Annotations map[string]string    `yaml:"annotations,omitempty"`
	Type        string               `yaml:"type,omitempty"`
	Ports       []*ServicePortConfig `yaml:"ports,omitempty"`
	ExternalIPs []string             `yaml:"externalIPs,omitempty"`
}

// ServicePortConfig holds the port configuration of a component service
type ServicePortConfig struct {
	Port          *int   `yaml:"port,omitempty"`
	ContainerPort *int   `yaml:"containerPort,omitempty"`
	Protocol      string `yaml:"protocol,omitempty"`
}

// IngressConfig holds the configuration of a component ingress
type IngressConfig struct {
	Name        string               `yaml:"name,omitempty"`
	Labels      map[string]string    `yaml:"labels,omitempty"`
	Annotations map[string]string    `yaml:"annotations,omitempty"`
	TLS         string               `yaml:"tls,omitempty"`
	Rules       []*IngressRuleConfig `yaml:"rules,omitempty"`
}

// IngressRuleConfig holds the port configuration of a component service
type IngressRuleConfig struct {
	Host        string `yaml:"host,omitempty"`
	TLS         string `yaml:"tls,omitempty"`
	Path        string `yaml:"path,omitempty"`
	ServicePort *int   `yaml:"servicePort,omitempty"`
	ServiceName string `yaml:"serviceName,omitempty"`
}

// AutoScalingConfig holds the autoscaling config of a component
type AutoScalingConfig struct {
	Horizontal *AutoScalingHorizontalConfig `yaml:"horizontal,omitempty"`
}

// AutoScalingHorizontalConfig holds the horizontal autoscaling config of a component
type AutoScalingHorizontalConfig struct {
	MaxReplicas           *int   `yaml:"maxReplicas,omitempty"`
	AverageCPU            string `yaml:"averageCPU,omitempty"`
	AverageRelativeCPU    string `yaml:"averageRelativeCPU,omitempty"`
	AverageMemory         string `yaml:"averageMemory,omitempty"`
	AverageRelativeMemory string `yaml:"averageRelativeMemory,omitempty"`
}

// RollingUpdateConfig holds the configuration for rolling updates
type RollingUpdateConfig struct {
	Enabled        *bool  `yaml:"enabled,omitempty"`
	MaxSurge       string `yaml:"maxSurge,omitempty"`
	MaxUnavailable string `yaml:"maxUnavailable,omitempty"`
	Partition      *int   `yaml:"partition,omitempty"`
}

// HelmConfig defines the specific helm options used during deployment
type HelmConfig struct {
	Chart            *ChartConfig                `yaml:"chart,omitempty"`
	ComponentChart   *bool                       `yaml:"componentChart,omitempty"`
	Values           map[interface{}]interface{} `yaml:"values,omitempty"`
	ValuesFiles      []string                    `yaml:"valuesFiles,omitempty"`
	ReplaceImageTags *bool                       `yaml:"replaceImageTags,omitempty"`
	Wait             bool                        `yaml:"wait,omitempty"`
	Atomic           bool                        `yaml:"atomic,omitempty"`
	CleanupOnFail    bool                        `yaml:"cleanupOnFail,omitempty"`
	Recreate         bool                        `yaml:"recreate,omitempty"`
	DisableHooks     bool                        `yaml:"disableHooks,omitempty"`
	Timeout          *int64                      `yaml:"timeout,omitempty"`
	Force            bool                        `yaml:"force,omitempty"`
	Driver           string                      `yaml:"driver,omitempty"`
	V2               bool                        `yaml:"v2,omitempty"`
	TillerNamespace  string                      `yaml:"tillerNamespace,omitempty"`
}

// ChartConfig defines the helm chart options
type ChartConfig struct {
	Name     string `yaml:"name,omitempty"`
	Version  string `yaml:"version,omitempty"`
	RepoURL  string `yaml:"repo,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// KubectlConfig defines the specific kubectl options used during deployment
type KubectlConfig struct {
	Manifests        []string `yaml:"manifests,omitempty"`
	Kustomize        *bool    `yaml:"kustomize,omitempty"`
	ReplaceImageTags *bool    `yaml:"replaceImageTags,omitempty"`
	Flags            []string `yaml:"flags,omitempty"`
	CmdPath          string   `yaml:"cmdPath,omitempty"`
}

// DevConfig defines the devspace deployment
type DevConfig struct {
	Ports       []*PortForwardingConfig `yaml:"ports,omitempty"`
	Open        []*OpenConfig           `yaml:"open,omitempty"`
	Sync        []*SyncConfig           `yaml:"sync,omitempty"`
	Logs        *LogsConfig             `yaml:"logs,omitempty"`
	AutoReload  *AutoReloadConfig       `yaml:"autoReload,omitempty"`
	Interactive *InteractiveConfig      `yaml:"interactive,omitempty"`
}

// PortForwardingConfig defines the ports for a port forwarding to a DevSpace
type PortForwardingConfig struct {
	ImageName     string            `yaml:"imageName,omitempty"`
	LabelSelector map[string]string `yaml:"labelSelector,omitempty"`
	Namespace     string            `yaml:"namespace,omitempty"`
	PortMappings  []*PortMapping    `yaml:"forward,omitempty"`
}

// PortMapping defines the ports for a PortMapping
type PortMapping struct {
	LocalPort   *int   `yaml:"port"`
	RemotePort  *int   `yaml:"remotePort,omitempty"`
	BindAddress string `yaml:"bindAddress,omitempty"`
}

// OpenConfig defines what to open after services have been started
type OpenConfig struct {
	URL string `yaml:"url,omitempty"`
}

// SyncConfig defines the paths for a SyncFolder
type SyncConfig struct {
	ImageName             string            `yaml:"imageName,omitempty"`
	LabelSelector         map[string]string `yaml:"labelSelector,omitempty"`
	ContainerName         string            `yaml:"containerName,omitempty"`
	Namespace             string            `yaml:"namespace,omitempty"`
	LocalSubPath          string            `yaml:"localSubPath,omitempty"`
	ContainerPath         string            `yaml:"containerPath,omitempty"`
	ExcludePaths          []string          `yaml:"excludePaths,omitempty"`
	DownloadExcludePaths  []string          `yaml:"downloadExcludePaths,omitempty"`
	UploadExcludePaths    []string          `yaml:"uploadExcludePaths,omitempty"`
	DownloadOnInitialSync *bool             `yaml:"downloadOnInitialSync,omitempty"`
	WaitInitialSync       *bool             `yaml:"waitInitialSync,omitempty"`
	BandwidthLimits       *BandwidthLimits  `yaml:"bandwidthLimits,omitempty"`
}

// BandwidthLimits defines the struct for specifying the sync bandwidth limits
type BandwidthLimits struct {
	Download *int64 `yaml:"download,omitempty"`
	Upload   *int64 `yaml:"upload,omitempty"`
}

// LogsConfig specifies the logs options for devspace dev
type LogsConfig struct {
	Disabled *bool    `yaml:"disabled,omitempty"`
	ShowLast *int     `yaml:"showLast,omitempty"`
	Images   []string `yaml:"images,omitempty"`
}

// AutoReloadConfig defines the struct for auto reloading devspace with additional paths
type AutoReloadConfig struct {
	Paths       []string `yaml:"paths,omitempty"`
	Deployments []string `yaml:"deployments,omitempty"`
	Images      []string `yaml:"images,omitempty"`
}

// InteractiveConfig defines the default interactive config
type InteractiveConfig struct {
	DefaultEnabled *bool                     `yaml:"defaultEnabled,omitempty"`
	Images         []*InteractiveImageConfig `yaml:"images,omitempty"`
	Terminal       *TerminalConfig           `yaml:"terminal,omitempty"`
}

// InteractiveImageConfig describes the interactive mode options for an image
type InteractiveImageConfig struct {
	Name       string   `yaml:"name,omitempty"`
	Entrypoint []string `yaml:"entrypoint,omitempty"`
	Cmd        []string `yaml:"cmd,omitempty"`
}

// TerminalConfig describes the terminal options
type TerminalConfig struct {
	ImageName     string            `yaml:"imageName,omitempty"`
	LabelSelector map[string]string `yaml:"labelSelector,omitempty"`
	ContainerName string            `yaml:"containerName,omitempty"`
	Namespace     string            `yaml:"namespace,omitempty"`
	Command       []string          `yaml:"command,omitempty"`
}

// DependencyConfig defines the devspace dependency
type DependencyConfig struct {
	Source             *SourceConfig `yaml:"source"`
	Profile            string        `yaml:"profile,omitempty"`
	SkipBuild          *bool         `yaml:"skipBuild,omitempty"`
	IgnoreDependencies *bool         `yaml:"ignoreDependencies,omitempty"`
	Namespace          string        `yaml:"namespace,omitempty"`
}

// SourceConfig defines the dependency source
type SourceConfig struct {
	Git      string `yaml:"git,omitempty"`
	SubPath  string `yaml:"subPath,omitempty"`
	Branch   string `yaml:"branch,omitempty"`
	Tag      string `yaml:"tag,omitempty"`
	Revision string `yaml:"revision,omitempty"`

	Path string `yaml:"path,omitempty"`
}

// HookConfig defines a hook
type HookConfig struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`

	When *HookWhenConfig `yaml:"when,omitempty"`
}

// HookWhenConfig defines when the hook should be executed
type HookWhenConfig struct {
	Before *HookWhenAtConfig `yaml:"before,omitempty"`
	After  *HookWhenAtConfig `yaml:"after,omitempty"`
}

// HookWhenAtConfig defines at which stage the hook should be executed
type HookWhenAtConfig struct {
	Images      string `yaml:"images,omitempty"`
	Deployments string `yaml:"deployments,omitempty"`
}

// CommandConfig defines the command specification
type CommandConfig struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}

// Variable describes the var definition
type Variable struct {
	Name              string          `yaml:"name"`
	Question          string          `yaml:"question,omitempty"`
	Options           []string        `yaml:"options,omitempty"`
	Password          bool            `yaml:"password,omitempty"`
	ValidationPattern string          `yaml:"validationPattern,omitempty"`
	ValidationMessage string          `yaml:"validationMessage,omitempty"`
	Default           string          `yaml:"default,omitempty"`
	Source            *VariableSource `yaml:"source,omitempty"`
}

// VariableSource is type of a variable source
type VariableSource string

// List of values that source can take
const (
	VariableSourceAll   VariableSource = "all"
	VariableSourceEnv   VariableSource = "env"
	VariableSourceInput VariableSource = "input"
)

// ProfileConfig defines a profile config
type ProfileConfig struct {
	Name    string         `yaml:"name"`
	Patches []*PatchConfig `yaml:"patches,omitempty"`
	Replace *ReplaceConfig `yaml:"replace,omitempty"`
}

// PatchConfig describes a config patch and how it should be applied
type PatchConfig struct {
	Operation string      `yaml:"op"`
	Path      string      `yaml:"path"`
	Value     interface{} `yaml:"value,omitempty"`
	From      string      `yaml:"from,omitempty"`
}

// ReplaceConfig defines a replace config that can override certain parts of the config completely
type ReplaceConfig struct {
	Images       map[string]*ImageConfig `yaml:"images,omitempty"`
	Deployments  []*DeploymentConfig     `yaml:"deployments,omitempty"`
	Dev          *DevConfig              `yaml:"dev,omitempty"`
	Dependencies []*DependencyConfig     `yaml:"dependencies,omitempty"`
	Hooks        []*HookConfig           `yaml:"hooks,omitempty"`
}
