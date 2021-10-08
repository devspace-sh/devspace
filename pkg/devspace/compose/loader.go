package compose

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	composeloader "github.com/compose-spec/compose-go/loader"
	composetypes "github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
)

var (
	DockerComposePaths = []string{"docker-compose.yaml", "docker-compose.yml"}
	DockerIgnorePath   = ".dockerignore"
	DefaultVolumeSize  = "5Gi"
)

type ConfigLoader interface {
	Load(log log.Logger) (*latest.Config, error)

	Save(config *latest.Config) error
}

type configLoader struct {
	composePath string
}

func GetDockerComposePath() string {
	for _, composePath := range DockerComposePaths {
		_, err := os.Stat(composePath)
		if err == nil {
			return composePath
		}
	}
	return ""
}

func NewDockerComposeLoader(composePath string) ConfigLoader {
	return &configLoader{
		composePath: composePath,
	}
}

func (d *configLoader) Load(log log.Logger) (*latest.Config, error) {
	composeFile, err := ioutil.ReadFile(d.composePath)
	if err != nil {
		return nil, err
	}

	dockerCompose, err := composeloader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{
				Content: composeFile,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	config := latest.New().(*latest.Config)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	var hooks []*latest.HookConfig
	var images map[string]*latest.ImageConfig
	deployments := []*latest.DeploymentConfig{}
	dev := latest.DevConfig{}
	baseDir := filepath.Dir(d.composePath)

	if len(dockerCompose.Networks) > 0 {
		log.Warn("networks are not supported")
	}

	dependentsMap, err := calculateDependentsMap(dockerCompose)
	if err != nil {
		return nil, err
	}

	err = dockerCompose.WithServices(nil, func(service composetypes.ServiceConfig) error {
		imageConfig, err := imageConfig(cwd, service)
		if err != nil {
			return err
		}
		if imageConfig != nil {
			if images == nil {
				images = map[string]*latest.ImageConfig{}
			}
			images[service.Name] = imageConfig
		}

		deploymentConfig, err := deploymentConfig(service, dockerCompose.Volumes, log)
		if err != nil {
			return err
		}
		deployments = append(deployments, deploymentConfig)

		err = addDevConfig(&dev, service, baseDir, log)
		if err != nil {
			return err
		}

		_, isDependency := dependentsMap[service.Name]
		if isDependency {
			waitHook := createWaitHook(service.Name)
			hooks = append(hooks, waitHook)
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	for secretName, secret := range dockerCompose.Secrets {
		createHook, err := createSecretHook(secretName, cwd, secret)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, createHook)
		hooks = append(hooks, deleteSecretHook(secretName))
	}

	config.Images = images
	config.Deployments = deployments
	config.Dev = dev
	config.Hooks = hooks

	return config, nil
}

func (d *configLoader) Save(config *latest.Config) error {
	// Convert to string
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Path to save the configuration to
	err = ioutil.WriteFile(constants.DefaultConfigPath, configYaml, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func addDevConfig(dev *latest.DevConfig, service composetypes.ServiceConfig, baseDir string, log log.Logger) error {
	devPorts := dev.Ports
	if devPorts == nil {
		devPorts = []*latest.PortForwardingConfig{}
	}

	if len(service.Ports) > 0 {
		portForwarding := &latest.PortForwardingConfig{
			LabelSelector: labelSelector(service.Name),
			PortMappings:  []*latest.PortMapping{},
		}
		for _, port := range service.Ports {
			portMapping := &latest.PortMapping{}

			if port.Published == 0 {
				log.Warnf("Unassigned port ranges are not supported: %s", port.Target)
				continue
			}

			if port.Published != port.Target {
				portMapping.LocalPort = ptr.Int(int(port.Published))
				portMapping.RemotePort = ptr.Int(int(port.Target))
			} else {
				portMapping.LocalPort = ptr.Int(int(port.Published))
			}

			if port.HostIP != "" {
				portMapping.BindAddress = port.HostIP
			}

			portForwarding.PortMappings = append(portForwarding.PortMappings, portMapping)
		}
		devPorts = append(devPorts, portForwarding)
	}

	if len(service.Expose) > 0 {
		portForwarding := &latest.PortForwardingConfig{
			LabelSelector: labelSelector(service.Name),
			PortMappings:  []*latest.PortMapping{},
		}
		for _, expose := range service.Expose {
			exposePort, err := strconv.Atoi(expose)
			if err != nil {
				return fmt.Errorf("expected integer for port number: %s", err.Error())
			}
			portForwarding.PortMappings = append(portForwarding.PortMappings, &latest.PortMapping{
				LocalPort: ptr.Int(exposePort),
			})
		}
		devPorts = append(devPorts, portForwarding)
	}

	devSync := dev.Sync
	if devSync == nil {
		devSync = []*latest.SyncConfig{}
	}

	for _, volume := range service.Volumes {
		if volume.Type == composetypes.VolumeTypeBind {
			localSubPath := volume.Source

			if strings.HasPrefix(localSubPath, "~") {
				localSubPath = fmt.Sprintf(`$!(echo "$HOME/%s")`, strings.TrimLeft(localSubPath, "~/"))
			}

			sync := &latest.SyncConfig{
				LabelSelector: labelSelector(service.Name),
				LocalSubPath:  localSubPath,
				ContainerPath: volume.Target,
			}

			_, err := os.Stat(filepath.Join(baseDir, volume.Source, DockerIgnorePath))
			if err == nil {
				sync.ExcludeFile = DockerIgnorePath
			}

			devSync = append(devSync, sync)
		}
	}

	if len(devPorts) > 0 {
		dev.Ports = devPorts
	}

	if len(devSync) > 0 {
		dev.Sync = devSync
	}

	return nil
}

func imageConfig(cwd string, service composetypes.ServiceConfig) (*latest.ImageConfig, error) {
	build := service.Build
	if build == nil {
		return nil, nil
	}

	context, err := filepath.Rel(cwd, filepath.Join(cwd, build.Context))
	if err != nil {
		return nil, err
	}

	dockerfile, err := filepath.Rel(cwd, filepath.Join(cwd, build.Context, build.Dockerfile))
	if err != nil {
		return nil, err
	}

	image := &latest.ImageConfig{
		Image:      resolveImage(service),
		Context:    filepath.ToSlash(context),
		Dockerfile: filepath.ToSlash(dockerfile),
	}

	buildOptions := &latest.BuildOptions{}
	hasBuildOptions := false
	if build.Args != nil {
		buildOptions.BuildArgs = build.Args
		hasBuildOptions = true
	}

	if build.Target != "" {
		buildOptions.Target = build.Target
		hasBuildOptions = true
	}

	if build.Network != "" {
		buildOptions.Network = build.Network
		hasBuildOptions = true
	}

	if hasBuildOptions {
		image.Build = &latest.BuildConfig{
			Docker: &latest.DockerConfig{
				Options: buildOptions,
			},
		}
	}

	return image, nil
}

func createSecretHook(name string, cwd string, secret composetypes.SecretConfig) (*latest.HookConfig, error) {
	file, err := filepath.Rel(cwd, filepath.Join(cwd, secret.File))
	if err != nil {
		return nil, err
	}

	return &latest.HookConfig{
		Events:  []string{"before:deploy"},
		Command: fmt.Sprintf("kubectl create secret generic %s --dry-run=client --from-file=%s=%s -o yaml | kubectl apply -f -", name, name, filepath.ToSlash(file)),
	}, nil
}

func deleteSecretHook(name string) *latest.HookConfig {
	return &latest.HookConfig{
		Events:  []string{"after:purge"},
		Command: fmt.Sprintf("kubectl delete secret %s --ignore-not-found", name),
	}
}

func deploymentConfig(service composetypes.ServiceConfig, composeVolumes map[string]composetypes.VolumeConfig, log log.Logger) (*latest.DeploymentConfig, error) {
	values := map[interface{}]interface{}{}

	volumes, volumeMounts := volumesConfig(service, composeVolumes, log)
	if len(volumes) > 0 {
		values["volumes"] = volumes
	}

	container, err := containerConfig(service, volumeMounts)
	if err != nil {
		return nil, err
	}
	values["containers"] = []interface{}{container}

	if service.Restart != "" {
		restartPolicy := string(v1.RestartPolicyNever)
		switch service.Restart {
		case "always":
			restartPolicy = string(v1.RestartPolicyAlways)
		case "on-failure":
			restartPolicy = string(v1.RestartPolicyOnFailure)
		}
		values["restartPolicy"] = restartPolicy
	}

	ports := []interface{}{}
	if len(service.Ports) > 0 {
		for _, port := range service.Ports {
			var protocol string
			switch port.Protocol {
			case "tcp":
				protocol = string(v1.ProtocolTCP)
			case "udp":
				protocol = string(v1.ProtocolUDP)
			default:
				return nil, fmt.Errorf("invalid protocol %s", port.Protocol)
			}

			if port.Published == 0 {
				log.Warnf("Unassigned port ranges are not supported: %s", port.Target)
				continue
			}

			ports = append(ports, map[interface{}]interface{}{
				"port":          int(port.Published),
				"containerPort": int(port.Target),
				"protocol":      protocol,
			})
		}
	}

	if len(service.Expose) > 0 {
		for _, port := range service.Expose {
			intPort, err := strconv.Atoi(port)
			if err != nil {
				return nil, fmt.Errorf("expected integer for port number: %s", err.Error())
			}
			ports = append(ports, map[interface{}]interface{}{
				"port": intPort,
			})
		}
	}

	if len(ports) > 0 {
		values["service"] = map[interface{}]interface{}{
			"ports": ports,
		}
	}

	if len(service.ExtraHosts) > 0 {
		hostsMap := map[string][]interface{}{}
		for _, host := range service.ExtraHosts {
			hostTokens := strings.Split(host, ":")
			hostName := hostTokens[0]
			hostIP := hostTokens[1]
			hostsMap[hostIP] = append(hostsMap[hostIP], hostName)
		}

		hostAliases := []interface{}{}
		for ip, hosts := range hostsMap {
			hostAliases = append(hostAliases, map[interface{}]interface{}{
				"ip":        ip,
				"hostnames": hosts,
			})
		}

		values["hostAliases"] = hostAliases
	}

	return &latest.DeploymentConfig{
		Name: service.Name,
		Helm: &latest.HelmConfig{
			ComponentChart: ptr.Bool(true),
			Values:         values,
		},
	}, nil
}

func volumesConfig(
	service composetypes.ServiceConfig,
	composeVolumes map[string]composetypes.VolumeConfig,
	log log.Logger,
) (volumes []interface{}, volumeMounts []interface{}) {
	for _, composeVolume := range composeVolumes {
		volumeName := resolveVolumeName(composeVolume)
		volume := createVolume(volumeName, DefaultVolumeSize)
		volumes = append(volumes, volume)
	}

	for _, secret := range service.Secrets {
		volume := createSecretVolume(secret)
		volumes = append(volumes, volume)

		volumeMount := createSecretVolumeMount(secret)
		volumeMounts = append(volumeMounts, volumeMount)
	}

	for _, serviceVolume := range service.Volumes {
		volumeName := resolveServiceVolumeName(service, serviceVolume, len(volumes)-1)

		switch serviceVolume.Type {
		case composetypes.VolumeTypeTmpfs:
			volume := createEmptyDirVolume(volumeName, serviceVolume)
			volumes = append(volumes, volume)

			volumeMount := createServiceVolumeMount(volumeName, serviceVolume)
			volumeMounts = append(volumeMounts, volumeMount)
		case composetypes.VolumeTypeVolume:
			if needsVolume(serviceVolume, composeVolumes) {
				volume := createVolume(volumeName, DefaultVolumeSize)
				volumes = append(volumes, volume)
			}

			volumeMount := createServiceVolumeMount(volumeName, serviceVolume)
			volumeMounts = append(volumeMounts, volumeMount)
		default:
			log.Warnf("%s volumes are not supported", serviceVolume.Type)
		}
	}

	return volumes, volumeMounts
}

func containerConfig(service composetypes.ServiceConfig, volumeMounts []interface{}) (map[interface{}]interface{}, error) {
	container := map[interface{}]interface{}{
		"image": resolveImage(service),
	}

	if service.ContainerName != "" {
		container["name"] = formatContainerName(service.ContainerName)
	}

	if len(service.Command) > 0 {
		container["args"] = shellCommandToSlice(service.Command)
	}

	if len(service.Entrypoint) > 0 {
		container["command"] = shellCommandToSlice(service.Entrypoint)
	}

	if service.Environment != nil {
		env := containerEnv(service.Environment)
		if len(env) > 0 {
			container["env"] = env
		}
	}

	if service.HealthCheck != nil {
		livenessProbe, err := containerLivenessProbe(service.HealthCheck)
		if err != nil {
			return nil, err
		}
		if livenessProbe != nil {
			container["livenessProbe"] = livenessProbe
		}
	}

	if len(volumeMounts) > 0 {
		container["volumeMounts"] = volumeMounts
	}

	return container, nil
}

func containerEnv(env composetypes.MappingWithEquals) []interface{} {
	envs := []interface{}{}
	keys := []string{}
	for name := range env {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		value := env[name]
		envs = append(envs, map[interface{}]interface{}{
			"name":  name,
			"value": *value,
		})
	}
	return envs
}

func containerLivenessProbe(health *composetypes.HealthCheckConfig) (map[interface{}]interface{}, error) {
	if len(health.Test) == 0 {
		return nil, nil
	}

	var command []interface{}
	testKind := health.Test[0]
	switch testKind {
	case "NONE":
		return nil, nil
	case "CMD":
		for _, test := range health.Test[1:] {
			command = append(command, test)
		}
	case "CMD-SHELL":
		command = append(command, "sh")
		command = append(command, "-c")
		command = append(command, health.Test[1])
	default:
		command = append(command, health.Test[0:])
	}

	livenessProbe := map[interface{}]interface{}{
		"exec": map[interface{}]interface{}{
			"command": command,
		},
	}

	if health.Retries != nil {
		livenessProbe["failureThreshold"] = int(*health.Retries)
	}

	if health.Interval != nil {
		period, err := time.ParseDuration(health.Interval.String())
		if err != nil {
			return nil, err
		}
		livenessProbe["periodSeconds"] = int(period.Seconds())
	}

	if health.StartPeriod != nil {
		initialDelay, err := time.ParseDuration(health.Interval.String())
		if err != nil {
			return nil, err
		}
		livenessProbe["initialDelaySeconds"] = int(initialDelay.Seconds())
	}

	return livenessProbe, nil
}

func createEmptyDirVolume(volumeName string, volume composetypes.ServiceVolumeConfig) interface{} {
	// create an emptyDir volume
	emptyDir := map[interface{}]interface{}{}
	if volume.Tmpfs != nil {
		emptyDir["sizeLimit"] = fmt.Sprintf("%d", volume.Tmpfs.Size)
	}
	return map[interface{}]interface{}{
		"name":     volumeName,
		"emptyDir": emptyDir,
	}
}

func createSecretVolume(secret composetypes.ServiceSecretConfig) interface{} {
	return map[interface{}]interface{}{
		"name": secret.Source,
		"secret": map[interface{}]interface{}{
			"secretName": secret.Source,
		},
	}
}

func createSecretVolumeMount(secret composetypes.ServiceSecretConfig) interface{} {
	target := secret.Source
	if secret.Target != "" {
		target = secret.Target
	}
	return map[interface{}]interface{}{
		"containerPath": fmt.Sprintf("/run/secrets/%s", target),
		"volume": map[interface{}]interface{}{
			"name":     secret.Source,
			"subPath":  target,
			"readOnly": true,
		},
	}
}

func createServiceVolumeMount(volumeName string, volume composetypes.ServiceVolumeConfig) interface{} {
	return map[interface{}]interface{}{
		"containerPath": volume.Target,
		"volume": map[interface{}]interface{}{
			"name":     volumeName,
			"readOnly": volume.ReadOnly,
		},
	}
}

func createVolume(name string, size string) interface{} {
	return map[interface{}]interface{}{
		"name": name,
		"size": size,
	}
}

func needsVolume(volume composetypes.ServiceVolumeConfig, composeVolumes map[string]composetypes.VolumeConfig) bool {
	if volume.Source == "" {
		return true
	}

	_, hasVolume := composeVolumes[volume.Source]
	return !hasVolume
}

func resolveImage(service composetypes.ServiceConfig) string {
	image := service.Name
	if service.Image != "" {
		image = service.Image
	}
	return image
}

func resolveServiceVolumeName(service composetypes.ServiceConfig, volume composetypes.ServiceVolumeConfig, idx int) string {
	volumeName := volume.Source
	if volumeName == "" {
		volumeName = fmt.Sprintf("%s-%d", service.Name, idx)
	}
	return volumeName
}

func resolveVolumeName(volume composetypes.VolumeConfig) string {
	return strings.TrimLeft(volume.Name, "_")
}

func createWaitHook(deploymentName string) *latest.HookConfig {
	return &latest.HookConfig{
		Events: []string{fmt.Sprintf("after:deploy:%s", deploymentName)},
		Container: &latest.HookContainer{
			LabelSelector: labelSelector(deploymentName),
		},
		Wait: &latest.HookWaitConfig{
			Running:            true,
			TerminatedWithCode: ptr.Int32(0),
		},
	}
}

func calculateDependentsMap(dockerCompose *composetypes.Project) (map[string][]string, error) {
	tree := map[string][]string{}
	err := dockerCompose.WithServices(nil, func(service composetypes.ServiceConfig) error {
		for _, name := range service.GetDependencies() {
			tree[name] = append(tree[name], service.Name)
		}
		return nil
	})
	return tree, err
}

func shellCommandToSlice(command composetypes.ShellCommand) []interface{} {
	var slice []interface{}
	for _, item := range command {
		slice = append(slice, item)
	}
	return slice
}

func formatContainerName(name string) string {
	return strings.Replace(name, "_", "-", -1)
}

func labelSelector(serviceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": serviceName,
	}
}
