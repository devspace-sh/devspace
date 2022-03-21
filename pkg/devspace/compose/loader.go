package compose

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	composeloader "github.com/compose-spec/compose-go/loader"
	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm"
	"github.com/loft-sh/devspace/pkg/util/log"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
)

var (
	DockerComposePaths         = []string{"docker-compose.yaml", "docker-compose.yml"}
	DockerIgnorePath           = ".dockerignore"
	DefaultVolumeSize          = "5Gi"
	UploadVolumesContainerName = "upload-volumes"
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

func (cl *configLoader) Load(log log.Logger) (*latest.Config, error) {
	composeFile, err := ioutil.ReadFile(cl.composePath)
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
	config.Name = dockerCompose.Name
	if config.Name == "" {
		config.Name = "docker-compose"
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// var hooks []*latest.HookConfig
	var images map[string]*latest.Image
	var deployments map[string]*latest.DeploymentConfig
	var dev map[string]*latest.DevPod
	baseDir := filepath.Dir(cl.composePath)

	if len(dockerCompose.Networks) > 0 {
		log.Warn("networks are not supported")
	}

	// dependentsMap, err := calculateDependentsMap(dockerCompose)
	// if err != nil {
	// 	return nil, err
	// }

	err = dockerCompose.WithServices(nil, func(service composetypes.ServiceConfig) error {
		imageConfig, err := imageConfig(cwd, service)
		if err != nil {
			return err
		}
		if imageConfig != nil {
			if images == nil {
				images = map[string]*latest.Image{}
			}
			images[service.Name] = imageConfig
		}

		deploymentName := formatName(service.Name)
		deploymentConfig, err := cl.deploymentConfig(service, dockerCompose.Volumes, log)
		if err != nil {
			return err
		}
		if deployments == nil {
			deployments = map[string]*latest.DeploymentConfig{}
		}
		deployments[deploymentName] = deploymentConfig

		devConfig, err := addDevConfig(service, baseDir, log)
		if err != nil {
			return err
		}
		if devConfig != nil {
			if dev == nil {
				dev = map[string]*latest.DevPod{}
			}
			dev[service.Name] = devConfig
		}

		// 	bindVolumeHooks := []*latest.HookConfig{}
		// 	for _, volume := range service.Volumes {
		// 		if volume.Type == composetypes.VolumeTypeBind {
		// 			bindVolumeHook := createUploadVolumeHook(service, volume)
		// 			bindVolumeHooks = append(bindVolumeHooks, bindVolumeHook)
		// 		}
		// 	}

		// 	if len(bindVolumeHooks) > 0 {
		// 		hooks = append(hooks, bindVolumeHooks...)
		// 		hooks = append(hooks, createUploadDoneHook(service))
		// 	}

		// 	_, isDependency := dependentsMap[service.Name]
		// 	if isDependency {
		// 		waitHook := createWaitHook(service)
		// 		hooks = append(hooks, waitHook)
		// 	}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// for secretName, secret := range dockerCompose.Secrets {
	// 	createHook, err := createSecretHook(secretName, cwd, secret)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	hooks = append(hooks, createHook)
	// 	hooks = append(hooks, deleteSecretHook(secretName))
	// }

	config.Images = images
	config.Deployments = deployments
	config.Dev = dev
	// config.Hooks = hooks

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

func addDevConfig(service composetypes.ServiceConfig, baseDir string, log log.Logger) (*latest.DevPod, error) {
	var dev *latest.DevPod

	devPorts := []*latest.PortMapping{}

	if len(service.Ports) > 0 {
		if dev == nil {
			dev = &latest.DevPod{
				LabelSelector: labelSelector(service.Name),
				Ports:         []*latest.PortMapping{},
			}
		}
		for _, port := range service.Ports {
			portMapping := &latest.PortMapping{}

			if port.Published == 0 {
				log.Warnf("Unassigned port ranges are not supported: %s", port.Target)
				continue
			}

			if port.Published != port.Target {
				portMapping.Port = fmt.Sprint(port.Published) + ":" + fmt.Sprint(port.Target)
			} else {
				portMapping.Port = fmt.Sprint(port.Published)
			}

			if port.HostIP != "" {
				portMapping.BindAddress = port.HostIP
			}

			devPorts = append(devPorts, portMapping)
		}
	}

	if len(service.Expose) > 0 {
		if dev == nil {
			dev = &latest.DevPod{
				LabelSelector: labelSelector(service.Name),
			}
		}

		for _, expose := range service.Expose {
			devPorts = append(devPorts, &latest.PortMapping{
				Port: expose,
			})
		}
	}

	// 	devSync := dev.Sync
	// 	if devSync == nil {
	// 		devSync = []*latest.SyncConfig{}
	// 	}

	// 	for _, volume := range service.Volumes {
	// 		if volume.Type == composetypes.VolumeTypeBind {
	// 			sync := &latest.SyncConfig{
	// 				LabelSelector: labelSelector(service.Name),
	// 				ContainerName: resolveContainerName(service),
	// 				LocalSubPath:  resolveLocalPath(volume),
	// 				ContainerPath: volume.Target,
	// 			}

	// 			_, err := os.Stat(filepath.Join(baseDir, volume.Source, DockerIgnorePath))
	// 			if err == nil {
	// 				sync.ExcludeFile = DockerIgnorePath
	// 			}

	// 			devSync = append(devSync, sync)
	// 		}
	// 	}

	if len(devPorts) > 0 {
		dev.Ports = devPorts
	}

	// 	if len(devSync) > 0 {
	// 		dev.Sync = devSync
	// 	}

	return dev, nil
}

func imageConfig(cwd string, service composetypes.ServiceConfig) (*latest.Image, error) {
	build := service.Build
	if build == nil {
		return nil, nil
	}

	context, err := filepath.Rel(cwd, filepath.Join(cwd, build.Context))
	if err != nil {
		return nil, err
	}
	context = filepath.ToSlash(context)
	if context == "." {
		context = ""
	}

	dockerfile, err := filepath.Rel(cwd, filepath.Join(cwd, build.Context, build.Dockerfile))
	if err != nil {
		return nil, err
	}

	image := &latest.Image{
		Image:      resolveImage(service),
		Context:    context,
		Dockerfile: filepath.ToSlash(dockerfile),
	}

	if build.Args != nil {
		image.BuildArgs = build.Args
	}

	if build.Target != "" {
		image.Target = build.Target
	}

	if build.Network != "" {
		image.Network = build.Network
	}

	// 	if hasBuildOptions {
	// 		image.Build = &latest.BuildConfig{
	// 			Docker: &latest.DockerConfig{
	// 				Options: buildOptions,
	// 			},
	// 		}
	// 	}

	if len(service.Entrypoint) > 0 {
		image.Entrypoint = service.Entrypoint
	}

	return image, nil
}

// func createSecretHook(name string, cwd string, secret composetypes.SecretConfig) (*latest.HookConfig, error) {
// 	file, err := filepath.Rel(cwd, filepath.Join(cwd, secret.File))
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &latest.HookConfig{
// 		Events:  []string{"before:deploy"},
// 		Command: fmt.Sprintf("kubectl create secret generic %s --namespace=${devspace.namespace} --dry-run=client --from-file=%s=%s -o yaml | kubectl apply -f -", name, name, filepath.ToSlash(file)),
// 	}, nil
// }

// func deleteSecretHook(name string) *latest.HookConfig {
// 	return &latest.HookConfig{
// 		Events:  []string{"after:purge"},
// 		Command: fmt.Sprintf("kubectl delete secret %s --namespace=${devspace.namespace} --ignore-not-found", name),
// 	}
// }

func (cl *configLoader) deploymentConfig(service composetypes.ServiceConfig, composeVolumes map[string]composetypes.VolumeConfig, log log.Logger) (*latest.DeploymentConfig, error) {
	values := map[string]interface{}{}

	// 	volumes, volumeMounts, bindVolumeMounts := volumesConfig(service, composeVolumes, log)
	// 	if len(volumes) > 0 {
	// 		values["volumes"] = volumes
	// 	}

	// 	if hasLocalSync(service) {
	// 		values["initContainers"] = []interface{}{initContainerConfig(service, bindVolumeMounts)}
	// 	}

	container, err := containerConfig(service, []interface{}{})
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

			ports = append(ports, map[string]interface{}{
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
			ports = append(ports, map[string]interface{}{
				"port": intPort,
			})
		}
	}

	if len(ports) > 0 {
		values["service"] = map[string]interface{}{
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
			hostAliases = append(hostAliases, map[string]interface{}{
				"ip":        ip,
				"hostnames": hosts,
			})
		}

		values["hostAliases"] = hostAliases
	}

	return &latest.DeploymentConfig{
		Helm: &latest.HelmConfig{
			Chart: &latest.ChartConfig{
				Name:    helm.DevSpaceChartConfig.Name,
				RepoURL: helm.DevSpaceChartConfig.RepoURL,
			},
			Values: values,
		},
	}, nil
}

// func volumesConfig(
// 	service composetypes.ServiceConfig,
// 	composeVolumes map[string]composetypes.VolumeConfig,
// 	log log.Logger,
// ) (volumes []interface{}, volumeMounts []interface{}, bindVolumeMounts []interface{}) {
// 	for _, secret := range service.Secrets {
// 		volume := createSecretVolume(secret)
// 		volumes = append(volumes, volume)

// 		volumeMount := createSecretVolumeMount(secret)
// 		volumeMounts = append(volumeMounts, volumeMount)
// 	}

// 	var volumeVolumes []composetypes.ServiceVolumeConfig
// 	var bindVolumes []composetypes.ServiceVolumeConfig
// 	var tmpfsVolumes []composetypes.ServiceVolumeConfig
// 	for _, serviceVolume := range service.Volumes {
// 		switch serviceVolume.Type {
// 		case composetypes.VolumeTypeBind:
// 			bindVolumes = append(bindVolumes, serviceVolume)
// 		case composetypes.VolumeTypeTmpfs:
// 			tmpfsVolumes = append(tmpfsVolumes, serviceVolume)
// 		case composetypes.VolumeTypeVolume:
// 			volumeVolumes = append(volumeVolumes, serviceVolume)
// 		default:
// 			log.Warnf("%s volumes are not supported", serviceVolume.Type)
// 		}
// 	}

// 	volumeMap := map[string]interface{}{}
// 	for idx, volumeVolume := range volumeVolumes {
// 		volumeName := resolveServiceVolumeName(service, volumeVolume, idx+1)
// 		_, ok := volumeMap[volumeName]
// 		if !ok {
// 			volume := createVolume(volumeName, DefaultVolumeSize)
// 			volumes = append(volumes, volume)
// 			volumeMap[volumeName] = volume
// 		}

// 		volumeMount := createServiceVolumeMount(volumeName, volumeVolume)
// 		volumeMounts = append(volumeMounts, volumeMount)
// 	}

// 	for _, tmpfsVolume := range tmpfsVolumes {
// 		volumeName := resolveServiceVolumeName(service, tmpfsVolume, len(volumes))
// 		volume := createEmptyDirVolume(volumeName, tmpfsVolume)
// 		volumes = append(volumes, volume)

// 		volumeMount := createServiceVolumeMount(volumeName, tmpfsVolume)
// 		volumeMounts = append(volumeMounts, volumeMount)
// 	}

// 	for idx, bindVolume := range bindVolumes {
// 		volumeName := fmt.Sprintf("volume-%d", idx+1)
// 		volume := createEmptyDirVolume(volumeName, bindVolume)
// 		volumes = append(volumes, volume)

// 		volumeMount := createServiceVolumeMount(volumeName, bindVolume)
// 		volumeMounts = append(volumeMounts, volumeMount)

// 		bindVolumeMount := createInitVolumeMount(volumeName, bindVolume)
// 		bindVolumeMounts = append(bindVolumeMounts, bindVolumeMount)
// 	}

// 	return volumes, volumeMounts, bindVolumeMounts

// }

func containerConfig(service composetypes.ServiceConfig, volumeMounts []interface{}) (map[string]interface{}, error) {
	container := map[string]interface{}{
		"name":  resolveContainerName(service),
		"image": resolveImage(service),
	}

	if len(service.Command) > 0 {
		container["args"] = shellCommandToSlice(service.Command)
	}

	if !hasBuild(service) && len(service.Entrypoint) > 0 {
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
		envs = append(envs, map[string]interface{}{
			"name":  name,
			"value": *value,
		})
	}
	return envs
}

func containerLivenessProbe(health *composetypes.HealthCheckConfig) (map[string]interface{}, error) {
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

	livenessProbe := map[string]interface{}{
		"exec": map[string]interface{}{
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

// func createEmptyDirVolume(volumeName string, volume composetypes.ServiceVolumeConfig) interface{} {
// 	// create an emptyDir volume
// 	emptyDir := map[string]interface{}{}
// 	if volume.Tmpfs != nil {
// 		emptyDir["sizeLimit"] = fmt.Sprintf("%d", volume.Tmpfs.Size)
// 	}
// 	return map[string]interface{}{
// 		"name":     volumeName,
// 		"emptyDir": emptyDir,
// 	}
// }

// func createSecretVolume(secret composetypes.ServiceSecretConfig) interface{} {
// 	return map[string]interface{}{
// 		"name": secret.Source,
// 		"secret": map[string]interface{}{
// 			"secretName": secret.Source,
// 		},
// 	}
// }

// func createSecretVolumeMount(secret composetypes.ServiceSecretConfig) interface{} {
// 	target := secret.Source
// 	if secret.Target != "" {
// 		target = secret.Target
// 	}
// 	return map[string]interface{}{
// 		"containerPath": fmt.Sprintf("/run/secrets/%s", target),
// 		"volume": map[string]interface{}{
// 			"name":     secret.Source,
// 			"subPath":  target,
// 			"readOnly": true,
// 		},
// 	}
// }

// func createServiceVolumeMount(volumeName string, volume composetypes.ServiceVolumeConfig) interface{} {
// 	return map[string]interface{}{
// 		"containerPath": volume.Target,
// 		"volume": map[string]interface{}{
// 			"name":     volumeName,
// 			"readOnly": volume.ReadOnly,
// 		},
// 	}
// }

// func createInitVolumeMount(volumeName string, volume composetypes.ServiceVolumeConfig) interface{} {
// 	return map[string]interface{}{
// 		"containerPath": volume.Target,
// 		"volume": map[string]interface{}{
// 			"name":     volumeName,
// 			"readOnly": false,
// 		},
// 	}
// }

// func createVolume(name string, size string) interface{} {
// 	return map[string]interface{}{
// 		"name": name,
// 		"size": size,
// 	}
// }

func formatName(name string) string {
	return regexp.MustCompile(`[\._]`).ReplaceAllString(name, "-")
}

// func initContainerConfig(service composetypes.ServiceConfig, volumeMounts []interface{}) map[string]interface{} {
// 	return map[string]interface{}{
// 		"name":    UploadVolumesContainerName,
// 		"image":   "alpine",
// 		"command": []interface{}{"sh"},
// 		"args": []interface{}{
// 			"-c",
// 			"while [ ! -f /tmp/done ]; do sleep 2; done",
// 		},
// 		"volumeMounts": volumeMounts,
// 	}
// }

func resolveContainerName(service composetypes.ServiceConfig) string {
	if service.ContainerName != "" {
		return formatName(service.ContainerName)
	}
	return fmt.Sprintf("%s-container", formatName(service.Name))
}

func resolveImage(service composetypes.ServiceConfig) string {
	image := service.Name
	if service.Image != "" {
		image = service.Image
	}
	return image
}

// func resolveLocalPath(volume composetypes.ServiceVolumeConfig) string {
// 	localSubPath := volume.Source

// 	if strings.HasPrefix(localSubPath, "~") {
// 		localSubPath = fmt.Sprintf(`$!(echo "$HOME/%s")`, strings.TrimLeft(localSubPath, "~/"))
// 	}
// 	return localSubPath
// }

// func resolveServiceVolumeName(service composetypes.ServiceConfig, volume composetypes.ServiceVolumeConfig, idx int) string {
// 	volumeName := volume.Source
// 	if volumeName == "" {
// 		volumeName = fmt.Sprintf("%s-%d", formatName(service.Name), idx)
// 	}
// 	return volumeName
// }

// func createWaitHook(service composetypes.ServiceConfig) *latest.HookConfig {
// 	serviceName := formatName(service.Name)
// 	return &latest.HookConfig{
// 		Events: []string{fmt.Sprintf("after:deploy:%s", serviceName)},
// 		Container: &latest.HookContainer{
// 			LabelSelector: labelSelector(serviceName),
// 			ContainerName: resolveContainerName(service),
// 		},
// 		Wait: &latest.HookWaitConfig{
// 			Running:            true,
// 			TerminatedWithCode: ptr.Int32(0),
// 		},
// 	}
// }

// func createUploadVolumeHook(service composetypes.ServiceConfig, volume composetypes.ServiceVolumeConfig) *latest.HookConfig {
// 	serviceName := formatName(service.Name)
// 	return &latest.HookConfig{
// 		Events: []string{"after:deploy:" + serviceName},
// 		Upload: &latest.HookSyncConfig{
// 			LocalPath:     resolveLocalPath(volume),
// 			ContainerPath: volume.Target,
// 		},
// 		Container: &latest.HookContainer{
// 			LabelSelector: labelSelector(service.Name),
// 			ContainerName: UploadVolumesContainerName,
// 		},
// 	}
// }

// func createUploadDoneHook(service composetypes.ServiceConfig) *latest.HookConfig {
// 	serviceName := formatName(service.Name)
// 	return &latest.HookConfig{
// 		Events:  []string{"after:deploy:" + serviceName},
// 		Command: "touch /tmp/done",
// 		Container: &latest.HookContainer{
// 			LabelSelector: labelSelector(service.Name),
// 			ContainerName: UploadVolumesContainerName,
// 		},
// 	}
// }

// func calculateDependentsMap(dockerCompose *composetypes.Project) (map[string][]string, error) {
// 	tree := map[string][]string{}
// 	err := dockerCompose.WithServices(nil, func(service composetypes.ServiceConfig) error {
// 		for _, name := range service.GetDependencies() {
// 			tree[name] = append(tree[name], service.Name)
// 		}
// 		return nil
// 	})
// 	return tree, err
// }

func shellCommandToSlice(command composetypes.ShellCommand) []interface{} {
	var slice []interface{}
	for _, item := range command {
		slice = append(slice, item)
	}
	return slice
}

func labelSelector(serviceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": serviceName,
	}
}

func hasBuild(service composetypes.ServiceConfig) bool {
	return service.Build != nil
}

// func hasLocalSync(service composetypes.ServiceConfig) bool {
// 	for _, volume := range service.Volumes {
// 		if volume.Type == composetypes.VolumeTypeBind {
// 			return true
// 		}
// 	}
// 	return false
// }
