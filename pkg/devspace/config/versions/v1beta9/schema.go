package v1beta9

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
)

// Version is the current api version
const Version string = "v1beta9"

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
	// Version holds the config version
	Version string `yaml:"version"`

	// Name specifies the name of the DevSpace project
	Name string `yaml:"name" json:"name"`

	// Images holds configuration of how devspace should build images
	Images map[string]*ImageConfig `yaml:"images,omitempty" json:"images,omitempty"`

	// Deployments is an ordered list of deployments to deploy via helm, kustomize or kubectl.
	Deployments []*DeploymentConfig `yaml:"deployments,omitempty" json:"deployments,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// Dev holds development configuration for the 'devspace dev' command.
	Dev *DevConfig `yaml:"dev,omitempty" json:"dev,omitempty"`

	// Dependencies are sub devspace projects that lie in a local folder or can be accessed via git
	Dependencies []*DependencyConfig `yaml:"dependencies,omitempty" json:"dependencies,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// Hooks are actions that are executed at certain points within the pipeline. Hooks are ordered and are executed
	// in the order they are specified.
	Hooks []*HookConfig `yaml:"hooks,omitempty" json:"hooks,omitempty"`

	// PullSecrets are image pull secrets that will be created by devspace in the target namespace
	// during devspace dev or devspace deploy
	PullSecrets []*PullSecretConfig `yaml:"pullSecrets,omitempty" json:"pullSecrets,omitempty" patchStrategy:"merge" patchMergeKey:"registry"`

	// Commands are custom commands that can be executed via 'devspace run COMMAND'
	Commands []*CommandConfig `yaml:"commands,omitempty" json:"commands,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// Vars are config variables that can be used inside other config sections to replace certain values dynamically
	Vars []*Variable `yaml:"vars,omitempty" json:"vars,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// Profiles can be used to change the current configuration and change the behaviour of devspace
	Profiles []*ProfileConfig `yaml:"profiles,omitempty" json:"profiles,omitempty"`
}

// ImageConfig defines the image specification
type ImageConfig struct {
	// Image is the complete image name including registry and repository
	// for example myregistry.com/mynamespace/myimage
	Image string `yaml:"image" json:"image"`

	// Tags is an array that specifes all tags that should be build during
	// the build process. If this is empty, devspace will generate a random tag
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// DEPRECATED: Use tags with ### instead
	// If TagsAppendRandom is true, for all tags defined for this image a random suffix in
	// the form of '-xxxxx' will be appended
	TagsAppendRandom bool `yaml:"tagsAppendRandom,omitempty" json:"tagsAppendRandom,omitempty"`

	// Specifies a path (relative or absolute) to the dockerfile
	Dockerfile string `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`

	// The context path to build with
	Context string `yaml:"context,omitempty" json:"context,omitempty"`

	// Entrypoint specifies an entrypoint that will be appended to the dockerfile during
	// image build in memory. Example: ["sleep", "99999"]
	Entrypoint []string `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`

	// Cmd specifies the arguments for the entrypoint that will be appended
	// during build in memory to the dockerfile
	Cmd []string `yaml:"cmd,omitempty" json:"cmd,omitempty"`

	// CreatePullSecret specifies if a pull secret should be created for this image in the
	// target namespace. Defaults to true
	CreatePullSecret *bool `yaml:"createPullSecret,omitempty" json:"createPullSecret,omitempty"`

	// DEPRECATED: Use rebuildStrategy instead
	// If this is true, devspace will not rebuild the image even though files have changed within
	// the context if a syncpath for this image is defined. This can reduce the number of builds
	// when running 'devspace dev'
	PreferSyncOverRebuild bool `yaml:"preferSyncOverRebuild,omitempty" json:"preferSyncOverRebuild,omitempty"`

	// If true injects a small restart script into the container and wraps the entrypoint of that
	// container, so that devspace is able to restart the complete container during sync.
	// Please make sure you either have an Entrypoint defined in the devspace config or in the
	// dockerfile for this image, otherwise devspace will fail.
	InjectRestartHelper bool `yaml:"injectRestartHelper,omitempty" json:"injectRestartHelper,omitempty"`

	// If specified DevSpace will load the restart helper from this location instead of using the bundled
	// one within DevSpace. Can be either a local path or an URL where to find the restart helper.
	RestartHelperPath string `yaml:"restartHelperPath,omitempty" json:"restartHelperPath,omitempty"`

	// These instructions will be appended to the Dockerfile that is build at the current build target
	// and are appended before the entrypoint and cmd instructions
	AppendDockerfileInstructions []string `yaml:"appendDockerfileInstructions,omitempty" json:"appendDockerfileInstructions,omitempty"`

	// RebuildStrategy is used to determine when DevSpace should rebuild an image. By default, devspace will
	// rebuild an image if one of the following conditions is true:
	// - The dockerfile has changed
	// - The configuration within the devspace.yaml for the image has changed
	// - A file within the docker context (excluding .dockerignore rules) has changed
	// This option is ignored for custom builds.
	RebuildStrategy RebuildStrategy `yaml:"rebuildStrategy,omitempty" json:"rebuildStrategy,omitempty"`

	// Specific build options how to build the specified image
	Build *BuildConfig `yaml:"build,omitempty" json:"build,omitempty"`
}

// RebuildStrategy is the type of a image rebuild strategy
type RebuildStrategy string

// List of values that source can take
const (
	RebuildStrategyDefault              RebuildStrategy = ""
	RebuildStrategyAlways               RebuildStrategy = "always"
	RebuildStrategyIgnoreContextChanges RebuildStrategy = "ignoreContextChanges"
)

// BuildConfig defines the build process for an image. Only one of the options below
// can be specified.
type BuildConfig struct {
	// If docker is specified, DevSpace will build the image using the local docker daemon
	Docker *DockerConfig `yaml:"docker,omitempty" json:"docker,omitempty"`

	// If kaniko is specified, DevSpace will build the image in-cluster with kaniko
	Kaniko *KanikoConfig `yaml:"kaniko,omitempty" json:"kaniko,omitempty"`

	// If buildKit is specified, DevSpace will build the image either in-cluster or locally with BuildKit
	BuildKit *BuildKitConfig `yaml:"buildKit,omitempty" json:"buildKit,omitempty"`

	// If custom is specified, devspace will build the image with the help of
	// a custom script.
	Custom *CustomConfig `yaml:"custom,omitempty" json:"custom,omitempty"`

	// This overrides other options and is able to disable the build for this image.
	// Useful if you just want to select the image in a sync path or via devspace enter --image
	Disabled *bool `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

// DockerConfig tells the DevSpace CLI to build with Docker on Minikube or on localhost
type DockerConfig struct {
	PreferMinikube  *bool         `yaml:"preferMinikube,omitempty" json:"preferMinikube,omitempty"`
	SkipPush        *bool         `yaml:"skipPush,omitempty" json:"skipPush,omitempty"`
	DisableFallback *bool         `yaml:"disableFallback,omitempty" json:"disableFallback,omitempty"`
	UseBuildKit     *bool         `yaml:"useBuildKit,omitempty" json:"useBuildKit,omitempty"`
	UseCLI          bool          `yaml:"useCli,omitempty" json:"useCli,omitempty"`
	Args            []string      `yaml:"args,omitempty" json:"args,omitempty"`
	Options         *BuildOptions `yaml:"options,omitempty" json:"options,omitempty"`
}

// BuildKitConfig tells the DevSpace CLI to
type BuildKitConfig struct {
	// If this is true, DevSpace will not push any images
	SkipPush *bool `yaml:"skipPush,omitempty" json:"skipPush,omitempty"`

	// If false, will not try to use the minikube docker daemon to build the image
	PreferMinikube *bool `yaml:"preferMinikube,omitempty" json:"preferMinikube,omitempty"`

	// If specified, DevSpace will use BuildKit to build the image within the cluster
	InCluster *BuildKitInClusterConfig `yaml:"inCluster,omitempty" json:"inCluster,omitempty"`

	// Additional arguments to call docker buildx build with
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`

	// Override the base command to create a builder and build images. Defaults to ["docker", "buildx"]
	Command []string `yaml:"command,omitempty" json:"command,omitempty"`

	// Additional build options
	Options *BuildOptions `yaml:"options,omitempty" json:"options,omitempty"`
}

// BuildKitInClusterConfig holds the buildkit builder config
type BuildKitInClusterConfig struct {
	// Name is the name of the builder to use. If omitted, DevSpace will try to create
	// or reuse a builder in the form devspace-$NAMESPACE
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Namespace where to create the builder deployment in. Defaults to the current
	// active namespace.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// If enabled will create a rootless builder deployment.
	Rootless bool `yaml:"rootless,omitempty" json:"rootless,omitempty"`

	// The docker image to use for the BuildKit deployment
	Image string `yaml:"image,omitempty" json:"image,omitempty"`

	// The node selector to use for the BuildKit deployment
	NodeSelector string `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`

	// By default, DevSpace will try to create a new builder if it cannot be found.
	// If this is true, DevSpace will fail if the specified builder cannot be found.
	NoCreate bool `yaml:"noCreate,omitempty" json:"noCreate,omitempty"`

	// By default, DevSpace will try to recreate the builder if the builder configuration
	// in the devspace.yaml differs from the actual builder configuration. If this is
	// true, DevSpace will not try to do that.
	NoRecreate bool `yaml:"noRecreate,omitempty" json:"noRecreate,omitempty"`

	// If enabled, DevSpace will not try to load the built image into the local docker
	// daemon if skip push is defined
	NoLoad bool `yaml:"noLoad,omitempty" json:"noLoad,omitempty"`

	// Additional args to create the builder with.
	CreateArgs []string `yaml:"createArgs,omitempty" json:"createArgs,omitempty"`
}

// KanikoConfig tells the DevSpace CLI to build with Docker on Minikube or on localhost
type KanikoConfig struct {
	// if a cache repository should be used. defaults to true
	Cache *bool `yaml:"cache,omitempty" json:"cache,omitempty"`

	// the snapshot mode kaniko should use. defaults to time
	SnapshotMode string `yaml:"snapshotMode,omitempty" json:"snapshotMode,omitempty"`

	// the image name of the kaniko pod to use
	Image string `yaml:"image,omitempty" json:"image,omitempty"`

	// the image to init the kaniko pod
	InitImage string `yaml:"initImage,omitempty" json:"initImage,omitempty"`

	// additional arguments that should be passed to kaniko
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`

	// replace the starting command for the kaniko container
	Command []string `yaml:"command,omitempty" json:"command,omitempty"`

	// the namespace where the kaniko pod should be run
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// if true pushing to insecure registries is allowed
	Insecure *bool `yaml:"insecure,omitempty" json:"insecure,omitempty"`

	// the pull secret to mount by default
	PullSecret string `yaml:"pullSecret,omitempty" json:"pullSecret,omitempty"`

	// If true will skip mounting the pull secret
	SkipPullSecretMount bool `yaml:"skipPullSecretMount,omitempty" json:"skipPullSecretMount,omitempty"`

	// the node selector to use for the kaniko pod
	NodeSelector map[string]string `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`

	// the service account to use for the kaniko pod
	ServiceAccount string `yaml:"serviceAccount,omitempty" json:"serviceAccount,omitempty"`

	// extra annotations that will be added to the build pod
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`

	// extra labels that will be added to the build pod
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`

	// extra environment variables that will be added to the build init container
	InitEnv map[string]string `yaml:"initEnv,omitempty" json:"initEnv,omitempty"`

	// extra environment variables that will be added to the build kaniko container
	// Will populate the env.value field.
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`

	// extra environment variables from configmap or secret that will be added to the build kaniko container
	// Will populate the env.valueFrom field.
	EnvFrom map[string]map[string]interface{} `yaml:"envFrom,omitempty" json:"envFrom,omitempty"`

	// additional mounts that will be added to the build pod
	AdditionalMounts []KanikoAdditionalMount `yaml:"additionalMounts,omitempty" json:"additionalMounts,omitempty"`

	// the resources that should be set on the kaniko pod
	Resources *KanikoPodResources `yaml:"resources,omitempty" json:"resources,omitempty"`

	// other build options that will be passed to the kaniko pod
	Options *BuildOptions `yaml:"options,omitempty" json:"options,omitempty"`
}

// KanikoPodResources describes the resources section of the started kaniko pod
type KanikoPodResources struct {
	// The requests part of the resources
	Requests map[string]string `yaml:"requests,omitempty" json:"requests,omitempty"`

	// The limits part of the resources
	Limits map[string]string `yaml:"limits,omitempty" json:"limits,omitempty"`
}

// KanikoAdditionalMount tells devspace how the additional mount of the kaniko pod should look like
type KanikoAdditionalMount struct {
	// The secret that should be mounted
	Secret *KanikoAdditionalMountSecret `yaml:"secret,omitempty" json:"secret,omitempty"`

	// The configMap that should be mounted
	ConfigMap *KanikoAdditionalMountConfigMap `yaml:"configMap,omitempty" json:"configMap,omitempty"`

	// Mounted read-only if true, read-write otherwise (false or unspecified).
	// Defaults to false.
	// +optional
	ReadOnly bool `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`

	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `yaml:"mountPath,omitempty" json:"mountPath,omitempty"`

	// Path within the volume from which the container's volume should be mounted.
	// Defaults to "" (volume's root).
	// +optional
	SubPath string `yaml:"subPath,omitempty" json:"subPath,omitempty"`
}

type KanikoAdditionalMountConfigMap struct {
	// Name of the configmap
	// +optional
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// If unspecified, each key-value pair in the Data field of the referenced
	// ConfigMap will be projected into the volume as a file whose name is the
	// key and content is the value. If specified, the listed keys will be
	// projected into the specified paths, and unlisted keys will not be
	// present. If a key is specified which is not present in the ConfigMap,
	// the volume setup will error unless it is marked optional. Paths must be
	// relative and may not contain the '..' path or start with '..'.
	// +optional
	Items []KanikoAdditionalMountKeyToPath `yaml:"items,omitempty" json:"items,omitempty"`

	// Optional: mode bits to use on created files by default. Must be a
	// value between 0 and 0777. Defaults to 0644.
	// Directories within the path are not affected by this setting.
	// This might be in conflict with other options that affect the file
	// mode, like fsGroup, and the result can be other mode bits set.
	// +optional
	DefaultMode *int32 `yaml:"defaultMode,omitempty" json:"defaultMode,omitempty"`
}

type KanikoAdditionalMountSecret struct {
	// Name of the secret in the pod's namespace to use.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#secret
	// +optional
	Name string `yaml:"name" json:"name"`

	// If unspecified, each key-value pair in the Data field of the referenced
	// Secret will be projected into the volume as a file whose name is the
	// key and content is the value. If specified, the listed keys will be
	// projected into the specified paths, and unlisted keys will not be
	// present. If a key is specified which is not present in the Secret,
	// the volume setup will error unless it is marked optional. Paths must be
	// relative and may not contain the '..' path or start with '..'.
	// +optional
	Items []KanikoAdditionalMountKeyToPath `yaml:"items,omitempty" json:"items,omitempty"`

	// Optional: mode bits to use on created files by default. Must be a
	// value between 0 and 0777. Defaults to 0644.
	// Directories within the path are not affected by this setting.
	// This might be in conflict with other options that affect the file
	// mode, like fsGroup, and the result can be other mode bits set.
	// +optional
	DefaultMode *int32 `yaml:"defaultMode,omitempty" json:"defaultMode,omitempty"`
}

type KanikoAdditionalMountKeyToPath struct {
	// The key to project.
	Key string `yaml:"key" json:"key"`

	// The relative path of the file to map the key to.
	// May not be an absolute path.
	// May not contain the path element '..'.
	// May not start with the string '..'.
	Path string `yaml:"path" json:"path"`

	// Optional: mode bits to use on this file, must be a value between 0
	// and 0777. If not specified, the volume defaultMode will be used.
	// This might be in conflict with other options that affect the file
	// mode, like fsGroup, and the result can be other mode bits set.
	// +optional
	Mode *int32 `yaml:"mode,omitempty" json:"mode,omitempty"`
}

// CustomConfig tells the DevSpace CLI to build with a custom build script
type CustomConfig struct {
	Command  string                `yaml:"command,omitempty" json:"command,omitempty"`
	Commands []CustomConfigCommand `yaml:"commands,omitempty" json:"commands,omitempty"`

	Args         []string `yaml:"args,omitempty" json:"args,omitempty"`
	AppendArgs   []string `yaml:"appendArgs,omitempty" json:"appendArgs,omitempty"`
	ImageFlag    string   `yaml:"imageFlag,omitempty" json:"imageFlag,omitempty"`
	ImageTagOnly bool     `yaml:"imageTagOnly,omitempty" json:"imageTagOnly,omitempty"`
	SkipImageArg bool     `yaml:"skipImageArg,omitempty" json:"skipImageArg,omitempty"`

	OnChange []string `yaml:"onChange,omitempty" json:"onChange,omitempty"`
}

// CustomConfigCommand holds the information about a command on a specific operating system
type CustomConfigCommand struct {
	Command         string `yaml:"command,omitempty" json:"command,omitempty"`
	OperatingSystem string `yaml:"os,omitempty" json:"os,omitempty"`
}

// BuildOptions defines options for building Docker images
type BuildOptions struct {
	Target    string             `yaml:"target,omitempty" json:"target,omitempty"`
	Network   string             `yaml:"network,omitempty" json:"network,omitempty"`
	BuildArgs map[string]*string `yaml:"buildArgs,omitempty" json:"buildArgs,omitempty"`
}

// DeploymentConfig defines the configuration how the devspace should be deployed
type DeploymentConfig struct {
	Name      string         `yaml:"name" json:"name"`
	Namespace string         `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Helm      *HelmConfig    `yaml:"helm,omitempty" json:"helm,omitempty"`
	Kubectl   *KubectlConfig `yaml:"kubectl,omitempty" json:"kubectl,omitempty"`
}

// ComponentConfig holds the component information
type ComponentConfig struct {
	InitContainers      []*ContainerConfig       `yaml:"initContainers,omitempty" json:"initContainers,omitempty"`
	Containers          []*ContainerConfig       `yaml:"containers,omitempty" json:"containers,omitempty"`
	Labels              map[string]string        `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations         map[string]string        `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Volumes             []*VolumeConfig          `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Service             *ServiceConfig           `yaml:"service,omitempty" json:"service,omitempty"`
	ServiceName         string                   `yaml:"serviceName,omitempty" json:"serviceName,omitempty"`
	Ingress             *IngressConfig           `yaml:"ingress,omitempty" json:"ingress,omitempty"`
	Replicas            *int                     `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	Autoscaling         *AutoScalingConfig       `yaml:"autoScaling,omitempty" json:"autoScaling,omitempty"`
	RollingUpdate       *RollingUpdateConfig     `yaml:"rollingUpdate,omitempty" json:"rollingUpdate,omitempty"`
	PullSecrets         []*string                `yaml:"pullSecrets,omitempty" json:"pullSecrets,omitempty"`
	Tolerations         []map[string]interface{} `yaml:"tolerations,omitempty" json:"tolerations,omitempty"`
	Affinity            map[string]interface{}   `yaml:"affinity,omitempty" json:"affinity,omitempty"`
	NodeSelector        map[string]interface{}   `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`
	NodeName            string                   `yaml:"nodeName,omitempty" json:"nodeName,omitempty"`
	PodManagementPolicy string                   `yaml:"podManagementPolicy,omitempty" json:"podManagementPolicy,omitempty"`

	DNSConfig                     map[string]interface{}   `yaml:"dnsConfig,omitempty" json:"dnsConfig,omitempty"`
	HostAliases                   []map[string]interface{} `yaml:"hostAliases,omitempty" json:"hostAliases,omitempty"`
	Overhead                      map[string]interface{}   `yaml:"overhead,omitempty" json:"overhead,omitempty"`
	ReadinessGates                []map[string]interface{} `yaml:"readinessGates,omitempty" json:"readinessGates,omitempty"`
	SecurityContext               map[string]interface{}   `yaml:"securityContext,omitempty" json:"securityContext,omitempty"`
	TopologySpreadConstraints     []map[string]interface{} `yaml:"topologySpreadConstraints,omitempty" json:"topologySpreadConstraints,omitempty"`
	ActiveDeadlineSeconds         *int                     `yaml:"activeDeadlineSeconds,omitempty" json:"activeDeadlineSeconds,omitempty"`
	AutomountServiceAccountToken  *bool                    `yaml:"automountServiceAccountToken,omitempty" json:"automountServiceAccountToken,omitempty"`
	DNSPolicy                     *string                  `yaml:"dnsPolicy,omitempty" json:"dnsPolicy,omitempty"`
	EnableServiceLinks            *bool                    `yaml:"enableServiceLinks,omitempty" json:"enableServiceLinks,omitempty"`
	HostIPC                       *bool                    `yaml:"hostIPC,omitempty" json:"hostIPC,omitempty"`
	HostNetwork                   *bool                    `yaml:"hostNetwork,omitempty" json:"hostNetwork,omitempty"`
	HostPID                       *bool                    `yaml:"hostPID,omitempty" json:"hostPID,omitempty"`
	Hostname                      *string                  `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	PreemptionPolicy              *string                  `yaml:"preemptionPolicy,omitempty" json:"preemptionPolicy,omitempty"`
	Priority                      *int                     `yaml:"priority,omitempty" json:"priority,omitempty"`
	PriorityClassName             *string                  `yaml:"priorityClassName,omitempty" json:"priorityClassName,omitempty"`
	RestartPolicy                 *string                  `yaml:"restartPolicy,omitempty" json:"restartPolicy,omitempty"`
	RuntimeClassName              *string                  `yaml:"runtimeClassName,omitempty" json:"runtimeClassName,omitempty"`
	SchedulerName                 *string                  `yaml:"schedulerName,omitempty" json:"schedulerName,omitempty"`
	ServiceAccount                *string                  `yaml:"serviceAccount,omitempty" json:"serviceAccount,omitempty"`
	ServiceAccountName            *string                  `yaml:"serviceAccountName,omitempty" json:"serviceAccountName,omitempty"`
	SetHostnameAsFQDN             *bool                    `yaml:"setHostnameAsFQDN,omitempty" json:"setHostnameAsFQDN,omitempty"`
	ShareProcessNamespace         *bool                    `yaml:"shareProcessNamespace,omitempty" json:"shareProcessNamespace,omitempty"`
	Subdomain                     *string                  `yaml:"subdomain,omitempty" json:"subdomain,omitempty"`
	TerminationGracePeriodSeconds *int                     `yaml:"terminationGracePeriodSeconds,omitempty" json:"terminationGracePeriodSeconds,omitempty"`
	EphemeralContainers           []map[string]interface{} `yaml:"ephemeralContainers,omitempty" json:"ephemeralContainers,omitempty"`
}

// ContainerConfig holds the configurations of a container
type ContainerConfig struct {
	Name                     string                   `yaml:"name,omitempty" json:"name,omitempty"`
	Image                    string                   `yaml:"image,omitempty" json:"image,omitempty"`
	Command                  []string                 `yaml:"command,omitempty" json:"command,omitempty"`
	Args                     []string                 `yaml:"args,omitempty" json:"args,omitempty"`
	Stdin                    bool                     `yaml:"stdin,omitempty" json:"stdin,omitempty"`
	TTY                      bool                     `yaml:"tty,omitempty" json:"tty,omitempty"`
	Env                      []map[string]interface{} `yaml:"env,omitempty" json:"env,omitempty"`
	EnvFrom                  []map[string]interface{} `yaml:"envFrom,omitempty" json:"envFrom,omitempty"`
	VolumeMounts             []*VolumeMountConfig     `yaml:"volumeMounts,omitempty" json:"volumeMounts,omitempty"`
	Resources                map[string]interface{}   `yaml:"resources,omitempty" json:"resources,omitempty"`
	LivenessProbe            map[string]interface{}   `yaml:"livenessProbe,omitempty" json:"livenessProbe,omitempty"`
	ReadinessProbe           map[string]interface{}   `yaml:"readinessProbe,omitempty" json:"readinessProbe,omitempty"`
	StartupProbe             map[string]interface{}   `yaml:"startupProbe,omitempty" json:"startupProbe,omitempty"`
	SecurityContext          map[string]interface{}   `yaml:"securityContext,omitempty" json:"securityContext,omitempty"`
	Lifecycle                map[string]interface{}   `yaml:"lifecycle,omitempty" json:"lifecycle,omitempty"`
	VolumeDevices            []map[string]interface{} `yaml:"volumeDevices,omitempty" json:"volumeDevices,omitempty"`
	ImagePullPolicy          string                   `yaml:"imagePullPolicy,omitempty" json:"imagePullPolicy,omitempty"`
	WorkingDir               string                   `yaml:"workingDir,omitempty" json:"workingDir,omitempty"`
	StdinOnce                bool                     `yaml:"stdinOnce,omitempty" json:"stdinOnce,omitempty"`
	TerminationMessagePath   string                   `yaml:"terminationMessagePath,omitempty" json:"terminationMessagePath,omitempty"`
	TerminationMessagePolicy string                   `yaml:"terminationMessagePolicy,omitempty" json:"terminationMessagePolicy,omitempty"`
}

// VolumeMountConfig holds the configuration for a specific mount path
type VolumeMountConfig struct {
	ContainerPath string                   `yaml:"containerPath,omitempty" json:"containerPath,omitempty"`
	Volume        *VolumeMountVolumeConfig `yaml:"volume,omitempty" json:"volume,omitempty"`
}

// VolumeMountVolumeConfig holds the configuration for a specific mount path volume
type VolumeMountVolumeConfig struct {
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
	SubPath  string `yaml:"subPath,omitempty" json:"subPath,omitempty"`
	ReadOnly *bool  `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`
}

// VolumeConfig holds the configuration for a specific volume
type VolumeConfig struct {
	Name             string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Labels           map[string]string      `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations      map[string]string      `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Size             string                 `yaml:"size,omitempty" json:"size,omitempty"`
	ConfigMap        map[string]interface{} `yaml:"configMap,omitempty" json:"configMap,omitempty"`
	Secret           map[string]interface{} `yaml:"secret,omitempty" json:"secret,omitempty"`
	StorageClassName string                 `yaml:"storageClassName,omitempty" json:"storageClassName,omitempty"`
	VolumeMode       string                 `yaml:"volumeMode,omitempty" json:"volumeMode,omitempty"`
	VolumeName       string                 `yaml:"volumeName,omitempty" json:"volumeName,omitempty"`
	DataSource       map[string]interface{} `yaml:"dataSource,omitempty" json:"dataSource,omitempty"`
	AccessModes      []string               `yaml:"accessModes,omitempty" json:"accessModes,omitempty"`
}

// ServiceConfig holds the configuration of a component service
type ServiceConfig struct {
	Name                     string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Labels                   map[string]string      `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations              map[string]string      `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Type                     string                 `yaml:"type,omitempty" json:"type,omitempty"`
	Ports                    []*ServicePortConfig   `yaml:"ports,omitempty" json:"ports,omitempty"`
	ExternalIPs              []string               `yaml:"externalIPs,omitempty" json:"externalIPs,omitempty"`
	ClusterIP                string                 `yaml:"clusterIP,omitempty" json:"clusterIP,omitempty"`
	ExternalName             string                 `yaml:"externalName,omitempty" json:"externalName,omitempty"`
	ExternalTrafficPolicy    string                 `yaml:"externalTrafficPolicy,omitempty" json:"externalTrafficPolicy,omitempty"`
	HealthCheckNodePort      int                    `yaml:"healthCheckNodePort,omitempty" json:"healthCheckNodePort,omitempty"`
	IPFamily                 *string                `yaml:"ipFamily,omitempty" json:"ipFamily,omitempty"`
	LoadBalancerIP           *string                `yaml:"loadBalancerIP,omitempty" json:"loadBalancerIP,omitempty"`
	LoadBalancerSourceRanges []string               `yaml:"loadBalancerSourceRanges,omitempty" json:"loadBalancerSourceRanges,omitempty"`
	PublishNotReadyAddresses bool                   `yaml:"publishNotReadyAddresses,omitempty" json:"publishNotReadyAddresses,omitempty"`
	SessionAffinity          map[string]interface{} `yaml:"sessionAffinity,omitempty" json:"sessionAffinity,omitempty"`
	SessionAffinityConfig    map[string]interface{} `yaml:"sessionAffinityConfig,omitempty" json:"sessionAffinityConfig,omitempty"`
	TopologyKeys             []string               `yaml:"topologyKeys,omitempty" json:"topologyKeys,omitempty"`
}

// ServicePortConfig holds the port configuration of a component service
type ServicePortConfig struct {
	Port          *int   `yaml:"port,omitempty" json:"port,omitempty"`
	ContainerPort *int   `yaml:"containerPort,omitempty" json:"containerPort,omitempty"`
	Protocol      string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
}

// IngressConfig holds the configuration of a component ingress
type IngressConfig struct {
	Name             string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Labels           map[string]string      `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations      map[string]string      `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	TLS              string                 `yaml:"tls,omitempty" json:"tls,omitempty"`
	TLSClusterIssuer string                 `yaml:"tlsClusterIssuer,omitempty" json:"tlsClusterIssuer,omitempty"`
	IngressClass     string                 `yaml:"ingressClass,omitempty" json:"ingressClass,omitempty"`
	Rules            []*IngressRuleConfig   `yaml:"rules,omitempty" json:"rules,omitempty"`
	Backend          map[string]interface{} `yaml:"backend,omitempty" json:"backend,omitempty"`
	IngressClassName *string                `yaml:"ingressClassName,omitempty" json:"ingressClassName,omitempty"`
}

// IngressRuleConfig holds the port configuration of a component service
type IngressRuleConfig struct {
	Host        string `yaml:"host,omitempty" json:"host,omitempty"`
	TLS         string `yaml:"tls,omitempty" json:"tls,omitempty"` // DEPRECATED
	Path        string `yaml:"path,omitempty" json:"path,omitempty"`
	ServicePort *int   `yaml:"servicePort,omitempty" json:"servicePort,omitempty"`
	ServiceName string `yaml:"serviceName,omitempty" json:"serviceName,omitempty"`
}

// AutoScalingConfig holds the autoscaling config of a component
type AutoScalingConfig struct {
	Horizontal *AutoScalingHorizontalConfig `yaml:"horizontal,omitempty" json:"horizontal,omitempty"`
}

// AutoScalingHorizontalConfig holds the horizontal autoscaling config of a component
type AutoScalingHorizontalConfig struct {
	MaxReplicas           *int   `yaml:"maxReplicas,omitempty" json:"maxReplicas,omitempty"`
	AverageCPU            string `yaml:"averageCPU,omitempty" json:"averageCPU,omitempty"`
	AverageRelativeCPU    string `yaml:"averageRelativeCPU,omitempty" json:"averageRelativeCPU,omitempty"`
	AverageMemory         string `yaml:"averageMemory,omitempty" json:"averageMemory,omitempty"`
	AverageRelativeMemory string `yaml:"averageRelativeMemory,omitempty" json:"averageRelativeMemory,omitempty"`
}

// RollingUpdateConfig holds the configuration for rolling updates
type RollingUpdateConfig struct {
	Enabled        *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MaxSurge       string `yaml:"maxSurge,omitempty" json:"maxSurge,omitempty"`
	MaxUnavailable string `yaml:"maxUnavailable,omitempty" json:"maxUnavailable,omitempty"`
	Partition      *int   `yaml:"partition,omitempty" json:"partition,omitempty"`
}

// HelmConfig defines the specific helm options used during deployment
type HelmConfig struct {
	Chart            *ChartConfig           `yaml:"chart,omitempty" json:"chart,omitempty"`
	ComponentChart   *bool                  `yaml:"componentChart,omitempty" json:"componentChart,omitempty"`
	Values           map[string]interface{} `yaml:"values,omitempty" json:"values,omitempty"`
	ValuesFiles      []string               `yaml:"valuesFiles,omitempty" json:"valuesFiles,omitempty"`
	ReplaceImageTags *bool                  `yaml:"replaceImageTags,omitempty" json:"replaceImageTags,omitempty"`
	Wait             bool                   `yaml:"wait,omitempty" json:"wait,omitempty"`
	DisplayOutput    bool                   `yaml:"displayOutput,omitempty" json:"output,omitempty"`
	Timeout          *int64                 `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Force            bool                   `yaml:"force,omitempty" json:"force,omitempty"`
	Atomic           bool                   `yaml:"atomic,omitempty" json:"atomic,omitempty"`
	CleanupOnFail    bool                   `yaml:"cleanupOnFail,omitempty" json:"cleanupOnFail,omitempty"`
	Recreate         bool                   `yaml:"recreate,omitempty" json:"recreate,omitempty"`
	DisableHooks     bool                   `yaml:"disableHooks,omitempty" json:"disableHooks,omitempty"`
	Driver           string                 `yaml:"driver,omitempty" json:"driver,omitempty"`
	Path             string                 `yaml:"path,omitempty" json:"path,omitempty"`
	V2               bool                   `yaml:"v2,omitempty" json:"v2,omitempty"`
	TillerNamespace  string                 `yaml:"tillerNamespace,omitempty" json:"tillerNamespace,omitempty"`

	DeleteArgs   []string `yaml:"deleteArgs,omitempty" json:"deleteArgs,omitempty"`
	TemplateArgs []string `yaml:"templateArgs,omitempty" json:"templateArgs,omitempty"`
	UpgradeArgs  []string `yaml:"upgradeArgs,omitempty" json:"upgradeArgs,omitempty"`
	FetchArgs    []string `yaml:"fetchArgs,omitempty" json:"fetchArgs,omitempty"`
}

// ChartConfig defines the helm chart options
type ChartConfig struct {
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
	Version  string `yaml:"version,omitempty" json:"version,omitempty"`
	RepoURL  string `yaml:"repo,omitempty" json:"repo,omitempty"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// KubectlConfig defines the specific kubectl options used during deployment
type KubectlConfig struct {
	Manifests        []string `yaml:"manifests,omitempty" json:"manifests,omitempty"`
	Kustomize        *bool    `yaml:"kustomize,omitempty" json:"kustomize,omitempty"`
	KustomizeArgs    []string `yaml:"kustomizeArgs,omitempty" json:"kustomizeArgs,omitempty"`
	ReplaceImageTags *bool    `yaml:"replaceImageTags,omitempty" json:"replaceImageTags,omitempty"`
	DeleteArgs       []string `yaml:"deleteArgs,omitempty" json:"deleteArgs,omitempty"`
	CreateArgs       []string `yaml:"createArgs,omitempty" json:"createArgs,omitempty"`
	ApplyArgs        []string `yaml:"applyArgs,omitempty" json:"applyArgs,omitempty"`
	CmdPath          string   `yaml:"cmdPath,omitempty" json:"cmdPath,omitempty"`
}

// DevConfig defines the devspace deployment
type DevConfig struct {
	Ports       []*PortForwardingConfig `yaml:"ports,omitempty" json:"ports,omitempty"`
	Open        []*OpenConfig           `yaml:"open,omitempty" json:"open,omitempty"`
	Sync        []*SyncConfig           `yaml:"sync,omitempty" json:"sync,omitempty" patchStrategy:"merge" patchMergeKey:"localSubPath"`
	Logs        *LogsConfig             `yaml:"logs,omitempty" json:"logs,omitempty"`
	AutoReload  *AutoReloadConfig       `yaml:"autoReload,omitempty" json:"autoReload,omitempty"`
	Interactive *InteractiveConfig      `yaml:"interactive,omitempty" json:"interactive,omitempty"`
}

// PortForwardingConfig defines the ports for a port forwarding to a DevSpace
type PortForwardingConfig struct {
	ImageName     string            `yaml:"imageName,omitempty" json:"imageName,omitempty"`
	LabelSelector map[string]string `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`
	ContainerName string            `yaml:"containerName,omitempty" json:"containerName,omitempty"`
	Namespace     string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// Target Container architecture to use for the devspacehelper (currently amd64 or arm64). Defaults to amd64
	Arch ContainerArchitecture `yaml:"arch,omitempty" json:"arch,omitempty"`

	PortMappings        []*PortMapping `yaml:"forward,omitempty" json:"forward,omitempty"`
	PortMappingsReverse []*PortMapping `yaml:"reverseForward,omitempty" json:"reverseForward,omitempty"`
}

// PortMapping defines the ports for a PortMapping
type PortMapping struct {
	LocalPort   *int   `yaml:"port" json:"port"`
	RemotePort  *int   `yaml:"remotePort,omitempty" json:"remotePort,omitempty"`
	BindAddress string `yaml:"bindAddress,omitempty" json:"bindAddress,omitempty"`
}

// OpenConfig defines what to open after services have been started
type OpenConfig struct {
	URL string `yaml:"url,omitempty" json:"url,omitempty"`
}

// SyncConfig defines the paths for a SyncFolder
type SyncConfig struct {
	ImageName            string               `yaml:"imageName,omitempty" json:"imageName,omitempty"`
	LabelSelector        map[string]string    `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`
	ContainerName        string               `yaml:"containerName,omitempty" json:"containerName,omitempty"`
	Namespace            string               `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	LocalSubPath         string               `yaml:"localSubPath,omitempty" json:"localSubPath,omitempty"`
	ContainerPath        string               `yaml:"containerPath,omitempty" json:"containerPath,omitempty"`
	ExcludePaths         []string             `yaml:"excludePaths,omitempty" json:"excludePaths,omitempty"`
	DownloadExcludePaths []string             `yaml:"downloadExcludePaths,omitempty" json:"downloadExcludePaths,omitempty"`
	UploadExcludePaths   []string             `yaml:"uploadExcludePaths,omitempty" json:"uploadExcludePaths,omitempty"`
	InitialSync          InitialSyncStrategy  `yaml:"initialSync,omitempty" json:"initialSync,omitempty"`
	InitialSyncCompareBy InitialSyncCompareBy `yaml:"initialSyncCompareBy,omitempty" json:"initialSyncCompareBy,omitempty"`

	DisableDownload *bool `yaml:"disableDownload,omitempty" json:"disableDownload,omitempty"`
	DisableUpload   *bool `yaml:"disableUpload,omitempty" json:"disableUpload,omitempty"`

	WaitInitialSync *bool            `yaml:"waitInitialSync,omitempty" json:"waitInitialSync,omitempty"`
	BandwidthLimits *BandwidthLimits `yaml:"bandwidthLimits,omitempty" json:"bandwidthLimits,omitempty"`

	// Target Container architecture to use for the devspacehelper (currently amd64 or arm64). Defaults to amd64
	Arch ContainerArchitecture `yaml:"arch,omitempty" json:"arch,omitempty"`

	// If greater zero, describes the amount of milliseconds to wait after each checked 100 files
	ThrottleChangeDetection *int64 `yaml:"throttleChangeDetection,omitempty" json:"throttleChangeDetection,omitempty"`

	OnUpload   *SyncOnUpload   `yaml:"onUpload,omitempty" json:"onUpload,omitempty"`
	OnDownload *SyncOnDownload `yaml:"onDownload,omitempty" json:"onDownload,omitempty"`
}

type ContainerArchitecture string

const (
	ContainerArchitectureAmd64 ContainerArchitecture = "amd64"
	ContainerArchitectureArm64 ContainerArchitecture = "arm64"
)

// SyncOnUpload defines the struct for the command that should be executed when files / folders are uploaded
type SyncOnUpload struct {
	// If true restart container will try to restart the container after a change has been made. Make sure that
	// images.*.injectRestartHelper is enabled for the container that should be restarted or the devspace-restart-helper
	// script is present in the container root folder.
	RestartContainer bool `yaml:"restartContainer,omitempty" json:"restartContainer,omitempty"`

	// Defines what commands should be executed on the container side if a change is uploaded and applied in the target
	// container
	ExecRemote *SyncExecCommand `yaml:"execRemote,omitempty" json:"execRemote,omitempty"`
}

// SyncOnDownload defines the struct for the command that should be executed when files / folders are downloaded
type SyncOnDownload struct {
	ExecLocal *SyncExecCommand `yaml:"execLocal,omitempty" json:"execLocal,omitempty"`
}

// SyncExecCommand holds the configuration of commands that should be executed when files / folders are change
type SyncExecCommand struct {
	Command string   `yaml:"command,omitempty" json:"command,omitempty"`
	Args    []string `yaml:"args,omitempty" json:"args,omitempty"`

	// OnFileChange is invoked after every file change. DevSpace will wait for the command to successfully finish
	// and then will continue to upload files & create folders
	OnFileChange *SyncCommand `yaml:"onFileChange,omitempty" json:"onFileChange,omitempty"`

	// OnDirCreate is invoked after every directory that is created. DevSpace will wait for the command to successfully finish
	// and then will continue to upload files & create folders
	OnDirCreate *SyncCommand `yaml:"onDirCreate,omitempty" json:"onDirCreate,omitempty"`

	// OnBatch executes the given command after a batch of changes has been processed. DevSpace will wait for the command to finish
	// and then will continue execution. This is useful for commands
	// that shouldn't be executed after every single change that may take a little bit longer like recompiling etc.
	OnBatch *SyncCommand `yaml:"onBatch,omitempty" json:"onBatch,omitempty"`
}

// SyncCommand holds a command definition
type SyncCommand struct {
	Command string   `yaml:"command,omitempty" json:"command,omitempty"`
	Args    []string `yaml:"args,omitempty" json:"args,omitempty"`
}

// InitialSyncStrategy is the type of a initial sync strategy
type InitialSyncStrategy string

// List of values that source can take
const (
	InitialSyncStrategyMirrorLocal  InitialSyncStrategy = "mirrorLocal"
	InitialSyncStrategyMirrorRemote InitialSyncStrategy = "mirrorRemote"
	InitialSyncStrategyPreferLocal  InitialSyncStrategy = "preferLocal"
	InitialSyncStrategyPreferRemote InitialSyncStrategy = "preferRemote"
	InitialSyncStrategyPreferNewest InitialSyncStrategy = "preferNewest"
	InitialSyncStrategyKeepAll      InitialSyncStrategy = "keepAll"
)

// InitialSyncCompareBy is the type of how a change should be determined during the initial sync
type InitialSyncCompareBy string

// List of values that compare by can take
const (
	InitialSyncCompareByMTime InitialSyncCompareBy = "mtime"
	InitialSyncCompareBySize  InitialSyncCompareBy = "size"
)

// BandwidthLimits defines the struct for specifying the sync bandwidth limits
type BandwidthLimits struct {
	Download *int64 `yaml:"download,omitempty" json:"download,omitempty"`
	Upload   *int64 `yaml:"upload,omitempty" json:"upload,omitempty"`
}

// LogsConfig specifies the logs options for devspace dev
type LogsConfig struct {
	Disabled  *bool          `yaml:"disabled,omitempty" json:"disabled,omitempty"`
	ShowLast  *int           `yaml:"showLast,omitempty" json:"showLast,omitempty"`
	Sync      *bool          `yaml:"sync,omitempty" json:"sync,omitempty"`
	Images    []string       `yaml:"images,omitempty" json:"images,omitempty"`
	Selectors []LogsSelector `yaml:"selectors,omitempty" json:"selectors,omitempty"`
}

// LogsSelector holds configuration how to select a log target
type LogsSelector struct {
	LabelSelector map[string]string `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`
	Namespace     string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	ContainerName string            `yaml:"containerName,omitempty" json:"containerName,omitempty"`
}

// AutoReloadConfig defines the struct for auto reloading devspace with additional paths
type AutoReloadConfig struct {
	Paths       []string `yaml:"paths,omitempty" json:"paths,omitempty"`
	Deployments []string `yaml:"deployments,omitempty" json:"deployments,omitempty"`
	Images      []string `yaml:"images,omitempty" json:"images,omitempty"`
}

// InteractiveConfig defines the default interactive config
type InteractiveConfig struct {
	DefaultEnabled *bool                     `yaml:"defaultEnabled,omitempty" json:"defaultEnabled,omitempty"`
	Images         []*InteractiveImageConfig `yaml:"images,omitempty" json:"images,omitempty"`
	Terminal       *TerminalConfig           `yaml:"terminal,omitempty" json:"terminal,omitempty"`
}

// InteractiveImageConfig describes the interactive mode options for an image
type InteractiveImageConfig struct {
	Name       string   `yaml:"name,omitempty" json:"name,omitempty"`
	Entrypoint []string `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	Cmd        []string `yaml:"cmd,omitempty" json:"cmd,omitempty"`
}

// TerminalConfig describes the terminal options
type TerminalConfig struct {
	ImageName     string            `yaml:"imageName,omitempty" json:"imageName,omitempty"`
	LabelSelector map[string]string `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`
	ContainerName string            `yaml:"containerName,omitempty" json:"containerName,omitempty"`
	Namespace     string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Command       []string          `yaml:"command,omitempty" json:"command,omitempty"`
	WorkDir       string            `yaml:"workDir,omitempty" json:"workDir,omitempty"`
}

// DependencyConfig defines the devspace dependency
type DependencyConfig struct {
	Name               string          `yaml:"name" json:"name"`
	Source             *SourceConfig   `yaml:"source" json:"source"`
	Profile            string          `yaml:"profile,omitempty" json:"profile,omitempty"`
	ProfileParents     []string        `yaml:"profileParents,omitempty" json:"profileParents,omitempty"`
	Vars               []DependencyVar `yaml:"vars,omitempty" json:"vars,omitempty"`
	SkipBuild          *bool           `yaml:"skipBuild,omitempty" json:"skipBuild,omitempty"`
	IgnoreDependencies *bool           `yaml:"ignoreDependencies,omitempty" json:"ignoreDependencies,omitempty"`
	Namespace          string          `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

// DependencyVar holds an override value for a config variable
type DependencyVar struct {
	// Name is the name of the variable
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Value is the value to override
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

// SourceConfig defines the dependency source
type SourceConfig struct {
	Git            string   `yaml:"git,omitempty" json:"git,omitempty"`
	CloneArgs      []string `yaml:"cloneArgs,omitempty" json:"cloneArgs,omitempty"`
	DisableShallow bool     `yaml:"disableShallow,omitempty" json:"disableShallow,omitempty"`
	SubPath        string   `yaml:"subPath,omitempty" json:"subPath,omitempty"`
	Branch         string   `yaml:"branch,omitempty" json:"branch,omitempty"`
	Tag            string   `yaml:"tag,omitempty" json:"tag,omitempty"`
	Revision       string   `yaml:"revision,omitempty" json:"revision,omitempty"`
	ConfigName     string   `yaml:"configName,omitempty" json:"configName,omitempty"`

	Path string `yaml:"path,omitempty" json:"path,omitempty"`
}

// HookConfig defines a hook
type HookConfig struct {
	// Command is the base command that is either executed locally or in a remote container.
	// Command is mutually exclusive with other hook actions. In the case this is defined
	// together with where.container, DevSpace will until the target container is running and
	// only then execute the command. If the container does not start in time, DevSpace will fail.
	Command string `yaml:"command" json:"command"`
	// Args are additional arguments passed together with the command to execute.
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`
	// If Upload is specified, DevSpace will upload certain local files or folders into a
	// remote container.
	Upload *HookSyncConfig `yaml:"upload,omitempty" json:"upload,omitempty"`
	// Same as Upload, but with this option DevSpace will download files or folders from
	// a remote container.
	Download *HookSyncConfig `yaml:"download,omitempty" json:"download,omitempty"`
	// If logs is defined will print the logs of the target container. This is useful for containers
	// that should finish like init containers or job pods. Otherwise this hook will never terminate.
	Logs *HookLogsConfig `yaml:"logs,omitempty" json:"logs,omitempty"`
	// If wait is defined the hook will wait until the matched pod or container is running or is terminated
	// with a certain exit code.
	Wait *HookWaitConfig `yaml:"wait,omitempty" json:"wait,omitempty"`

	// If an operating system is defined, the hook will only be executed for the given os.
	// All supported golang OS types are supported and multiple can be combined with ','.
	OperatingSystem string `yaml:"os,omitempty" json:"os,omitempty"`

	// If true, the hook will be executed in the background.
	Background bool `yaml:"background,omitempty" json:"background,omitempty"`
	// If true, the hook will not output anything to the standard out of DevSpace except
	// for the case when the hook fails, where DevSpace will show the error including
	// the captured output streams of the hook.
	Silent bool `yaml:"silent,omitempty" json:"silent,omitempty"`

	// Specifies where the hook should be run. If this is omitted DevSpace expects a
	// local command hook.
	Where HookWhereConfig `yaml:"where,omitempty" json:"where,omitempty"`
	// Specifies when the hook should be run.
	When *HookWhenConfig `yaml:"when,omitempty" json:"when,omitempty"`
}

// HookWaitConfig defines a hook wait config
type HookWaitConfig struct {
	// If running is true, will wait until the matched containers are running. Can be used together with terminatedWithCode.
	Running bool `yaml:"running,omitempty" json:"running,omitempty"`

	// If terminatedWithCode is not nil, will wait until the matched containers are terminated with the given exit code.
	// If the container has exited with a different exit code, the hook will fail. Can be used together with running.
	TerminatedWithCode *int32 `yaml:"terminatedWithCode,omitempty" json:"terminatedWithCode,omitempty"`

	// The amount of seconds to wait until the hook will fail. Defaults to 150 seconds.
	Timeout int64 `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// HookLogsConfig defines a hook logs config
type HookLogsConfig struct {
	// If set, the number of lines from the end of the logs to show. If not specified,
	// logs are shown from the creation of the container
	TailLines *int64 `yaml:"tailLines,omitempty" json:"tailLines,omitempty"`
}

// HookSyncConfig defines a hook upload config
type HookSyncConfig struct {
	LocalPath     string `yaml:"localPath,omitempty" json:"localPath,omitempty"`
	ContainerPath string `yaml:"containerPath,omitempty" json:"containerPath,omitempty"`
}

// HookWhereConfig defines where to execute the hook
type HookWhereConfig struct {
	Container *HookContainer `yaml:"container,omitempty" json:"container,omitempty"`
}

// HookContainer defines how to select one or more containers to execute a hook in
type HookContainer struct {
	LabelSelector map[string]string `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`
	Pod           string            `yaml:"pod,omitempty" json:"pod,omitempty"`
	Namespace     string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	ImageName     string            `yaml:"imageName,omitempty" json:"imageName,omitempty"`
	ContainerName string            `yaml:"containerName,omitempty" json:"containerName,omitempty"`

	Wait    *bool `yaml:"wait,omitempty" json:"wait,omitempty"`
	Timeout int64 `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// HookWhenConfig defines when the hook should be executed
type HookWhenConfig struct {
	Before  *HookWhenAtConfig `yaml:"before,omitempty" json:"before,omitempty"`
	After   *HookWhenAtConfig `yaml:"after,omitempty" json:"after,omitempty"`
	OnError *HookWhenAtConfig `yaml:"onError,omitempty" json:"onError,omitempty"`
}

// HookWhenAtConfig defines at which stage the hook should be executed
type HookWhenAtConfig struct {
	Images           string `yaml:"images,omitempty" json:"images,omitempty"`
	PurgeDeployments string `yaml:"purgeDeployments,omitempty" json:"purgeDeployments,omitempty"`
	Deployments      string `yaml:"deployments,omitempty" json:"deployments,omitempty"`
	Dependencies     string `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	PullSecrets      string `yaml:"pullSecrets,omitempty" json:"pullSecrets,omitempty"`
}

// CommandConfig defines the command specification
type CommandConfig struct {
	Name        string `yaml:"name" json:"name"`
	Command     string `yaml:"command" json:"command"`
	Description string `yaml:"description" json:"description"`
}

// Variable describes the var definition
type Variable struct {
	Name              string         `yaml:"name" json:"name"`
	Question          string         `yaml:"question,omitempty" json:"question,omitempty"`
	Options           []string       `yaml:"options,omitempty" json:"options,omitempty"`
	Password          bool           `yaml:"password,omitempty" json:"password,omitempty"`
	ValidationPattern string         `yaml:"validationPattern,omitempty" json:"validationPattern,omitempty"`
	ValidationMessage string         `yaml:"validationMessage,omitempty" json:"validationMessage,omitempty"`
	NoCache           bool           `yaml:"noCache,omitempty" json:"noCache,omitempty"`
	Default           interface{}    `yaml:"default,omitempty" json:"default,omitempty"`
	Source            VariableSource `yaml:"source,omitempty" json:"source,omitempty"`

	Command  string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args     []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Commands []VariableCommand `yaml:"commands,omitempty" json:"commands,omitempty"`
}

type VariableCommand struct {
	Command         string   `yaml:"command,omitempty" json:"command,omitempty"`
	Args            []string `yaml:"args,omitempty" json:"args,omitempty"`
	OperatingSystem string   `yaml:"os,omitempty" json:"os,omitempty"`
}

// VariableSource is type of a variable source
type VariableSource string

// List of values that source can take
const (
	VariableSourceDefault VariableSource = ""
	VariableSourceAll     VariableSource = "all"
	VariableSourceEnv     VariableSource = "env"
	VariableSourceInput   VariableSource = "input"
	VariableSourceCommand VariableSource = "command"
	VariableSourceNone    VariableSource = "none"
)

// ProfileConfig defines a profile config
type ProfileConfig struct {
	Name           string                  `yaml:"name" json:"name"`
	Description    string                  `yaml:"description,omitempty" json:"description,omitempty"`
	Parent         string                  `yaml:"parent,omitempty" json:"parent,omitempty"`
	Parents        []*ProfileParent        `yaml:"parents,omitempty" json:"parents,omitempty"`
	Patches        []*PatchConfig          `yaml:"patches,omitempty" json:"patches,omitempty"`
	Replace        *ProfileConfigStructure `yaml:"replace,omitempty" json:"replace,omitempty"`
	Merge          *ProfileConfigStructure `yaml:"merge,omitempty" json:"merge,omitempty"`
	StrategicMerge *ProfileConfigStructure `yaml:"strategicMerge,omitempty" json:"strategicMerge,omitempty"`
}

// ProfileConfigStructure is the base structure used to validate profiles
type ProfileConfigStructure struct {
	Images       *map[string]interface{} `yaml:"images,omitempty" json:"images,omitempty"`
	Deployments  *[]interface{}          `yaml:"deployments,omitempty" json:"deployments,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	Dev          *map[string]interface{} `yaml:"dev,omitempty" json:"dev,omitempty"`
	Dependencies *[]interface{}          `yaml:"dependencies,omitempty" json:"dependencies,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	Hooks        *[]interface{}          `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	PullSecrets  *[]interface{}          `yaml:"pullSecrets,omitempty" json:"pullSecrets,omitempty" patchStrategy:"merge" patchMergeKey:"registry"`
	Commands     *[]interface{}          `yaml:"commands,omitempty" json:"commands,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	Vars         *[]interface{}          `yaml:"vars,omitempty" json:"vars,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
}

// ProfileParent defines where to load the profile from
type ProfileParent struct {
	Source  *SourceConfig `yaml:"source,omitempty" json:"source,omitempty"`
	Profile string        `yaml:"profile" json:"profile"`
}

// PatchConfig describes a config patch and how it should be applied
type PatchConfig struct {
	Operation string      `yaml:"op" json:"op"`
	Path      string      `yaml:"path" json:"path"`
	Value     interface{} `yaml:"value,omitempty" json:"value,omitempty"`
	From      string      `yaml:"from,omitempty" json:"from,omitempty"`
}

// PullSecretConfig defines a pull secret that should be created by DevSpace
type PullSecretConfig struct {
	// The registry to create the image pull secret for.
	// e.g. gcr.io
	Registry string `yaml:"registry" json:"registry"`

	// The username of the registry. If this is empty, devspace will try
	// to receive the auth data from the local docker
	Username string `yaml:"username,omitempty" json:"username,omitempty"`

	// The password to use for the registry. If this is empty, devspace will
	// try to receive the auth data from the local docker
	Password string `yaml:"password,omitempty" json:"password,omitempty"`

	// The optional email to use
	Email string `yaml:"email,omitempty" json:"email,omitempty"`

	// The secret to create
	Secret string `yaml:"secret,omitempty" json:"secret,omitempty"`

	// The service account to add the secret to
	ServiceAccounts []string `yaml:"serviceAccounts,omitempty" json:"serviceAccounts,omitempty"`
}
