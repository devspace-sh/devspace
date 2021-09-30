package compose

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

	parsed, err := composeloader.ParseYAML(composeFile)
	if err != nil {
		return nil, err
	}

	dockerCompose, err := composeloader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{
				Config: parsed,
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

	images := map[string]*latest.ImageConfig{}
	deployments := []*latest.DeploymentConfig{}
	dev := latest.DevConfig{}
	hooks := []*latest.HookConfig{}
	baseDir := filepath.Dir(d.composePath)

	for _, service := range dockerCompose.Services {
		imageConfig, err := imageConfig(cwd, service)
		if err != nil {
			return nil, err
		}
		if imageConfig != nil {
			images[service.Name] = imageConfig
		}

		deploymentConfig, err := deploymentConfig(service, dockerCompose.Volumes)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deploymentConfig)

		err = addDevConfig(&dev, service, baseDir)
		if err != nil {
			return nil, err
		}
	}

	for secretName, secret := range dockerCompose.Secrets {
		createHook, err := createSecretHook(secretName, cwd, secret)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, createHook)

		deleteHook, err := deleteSecretHook(secretName, cwd, secret)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, deleteHook)
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

	return &latest.ImageConfig{
		Image:      resolveImage(service),
		Context:    context,
		Dockerfile: dockerfile,
	}, nil
}

func createSecretHook(name string, cwd string, secret composetypes.SecretConfig) (*latest.HookConfig, error) {
	file, err := filepath.Rel(cwd, filepath.Join(cwd, secret.File))
	if err != nil {
		return nil, err
	}

	return &latest.HookConfig{
		Events:  []string{"before:deploy"},
		Command: fmt.Sprintf("kubectl create secret generic %s --dry-run=client --from-file=%s=%s -o yaml | kubectl apply -f -", name, name, file),
	}, nil
}

func deleteSecretHook(name string, cwd string, secret composetypes.SecretConfig) (*latest.HookConfig, error) {
	return &latest.HookConfig{
		Events:  []string{"after:purge"},
		Command: fmt.Sprintf("kubectl delete secret %s --ignore-not-found", name),
	}, nil
}

func deploymentConfig(service composetypes.ServiceConfig, volumesConfig map[string]composetypes.VolumeConfig) (*latest.DeploymentConfig, error) {
	values := map[interface{}]interface{}{}

	container, err := containerConfig(service)
	if err != nil {
		return nil, err
	}
	values["containers"] = []map[interface{}]interface{}{container}

	if service.Restart != "" {
		restartPolicy := v1.RestartPolicyNever
		switch service.Restart {
		case "always":
			restartPolicy = v1.RestartPolicyAlways
		case "on-failure":
			restartPolicy = v1.RestartPolicyOnFailure
		}
		values["restartPolicy"] = restartPolicy
	}

	ports := []map[string]interface{}{}
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

	volumes := []map[string]interface{}{}

	for _, volume := range service.Volumes {
		if volume.Type == composetypes.VolumeTypeVolume {
			deploymentVolume := map[string]interface{}{
				"name": volume.Source,
				"size": DefaultVolumeSize,
			}
			volumes = append(volumes, deploymentVolume)
		}
	}

	for _, secret := range service.Secrets {
		secretVolume := map[string]interface{}{
			"name": secret.Source,
			"secret": map[string]string{
				"secretName": secret.Source,
			},
		}
		volumes = append(volumes, secretVolume)
	}

	if len(volumes) > 0 {
		values["volumes"] = volumes
	}

	return &latest.DeploymentConfig{
		Name: service.Name,
		Helm: &latest.HelmConfig{
			ComponentChart: ptr.Bool(true),
			Values:         values,
		},
	}, nil
}

func containerConfig(service composetypes.ServiceConfig) (map[interface{}]interface{}, error) {
	container := map[interface{}]interface{}{
		"image": resolveImage(service),
	}

	if len(service.Command) > 0 {
		container["args"] = service.Command
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

	volumes := []map[string]interface{}{}

	for _, volume := range service.Volumes {
		if volume.Type == composetypes.VolumeTypeVolume {
			volumeMount := map[string]interface{}{
				"containerPath": volume.Target,
				"volume": map[string]interface{}{
					"name":     volume.Source,
					"readOnly": false,
				},
			}
			volumes = append(volumes, volumeMount)
		}
	}

	for _, secret := range service.Secrets {
		volumeMount := map[string]interface{}{
			"containerPath": fmt.Sprintf("/run/secrets/%s", secret.Source),
			"volume": map[string]interface{}{
				"name":     secret.Source,
				"subPath":  secret.Source,
				"readOnly": true,
			},
		}
		volumes = append(volumes, volumeMount)
	}

	if len(volumes) > 0 {
		container["volumeMounts"] = volumes
	}

	return container, nil
}

func containerEnv(env composetypes.MappingWithEquals) []map[string]string {
	envs := []map[string]string{}
	for name, value := range env {
		envs = append(envs, map[string]string{
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
	livenessProbe := map[string]interface{}{}

	testKind := health.Test[0]
	switch testKind {
	case "NONE":
		return nil, nil
	case "CMD":
		livenessProbe["exec"] = map[string][]string{
			"command": health.Test[1:],
		}
	case "CMD-SHELL":
		livenessProbe["exec"] = map[string][]string{
			"command": {"sh", "-c", health.Test[1]},
		}
	default:
		livenessProbe["exec"] = map[string][]string{
			"command": health.Test[0:],
		}
	}

	if health.Retries != nil {
		livenessProbe["failureThreshold"] = health.Retries
	}

	if health.Interval != nil {
		period, err := time.ParseDuration(health.Interval.String())
		if err != nil {
			return nil, err
		}
		livenessProbe["periodSeconds"] = period.Seconds()
	}

	if health.StartPeriod != nil {
		initialDelay, err := time.ParseDuration(health.Interval.String())
		if err != nil {
			return nil, err
		}
		livenessProbe["initialDelaySeconds"] = initialDelay.Seconds()
	}

	return livenessProbe, nil
}

func resolveImage(service composetypes.ServiceConfig) string {
	image := service.Name
	if service.Image != "" {
		image = service.Image
	}
	return image
}

func addDevConfig(dev *latest.DevConfig, service composetypes.ServiceConfig, baseDir string) error {
	devPorts := dev.Ports
	if devPorts == nil {
		devPorts = []*latest.PortForwardingConfig{}
	}

	if len(service.Ports) > 0 {
		portForwarding := &latest.PortForwardingConfig{
			ImageSelector: resolveImage(service),
			PortMappings:  []*latest.PortMapping{},
		}
		for _, port := range service.Ports {
			portMapping := &latest.PortMapping{}

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
			ImageSelector: resolveImage(service),
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
				ImageSelector: resolveImage(service),
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
