package compose

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	v1 "k8s.io/api/core/v1"
)

func (cb *configBuilder) AddDeployment(dockerCompose *composetypes.Project, service composetypes.ServiceConfig) error {
	values := map[string]interface{}{}

	volumes, volumeMounts, _ := volumesConfig(service, dockerCompose.Volumes, cb.log)
	if len(volumes) > 0 {
		values["volumes"] = volumes
	}

	container, err := containerConfig(service, volumeMounts)
	if err != nil {
		return err
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
	for _, port := range service.Ports {
		var protocol string
		switch port.Protocol {
		case "tcp":
			protocol = string(v1.ProtocolTCP)
		case "udp":
			protocol = string(v1.ProtocolUDP)
		default:
			return fmt.Errorf("invalid protocol %s", port.Protocol)
		}

		if port.Published == "" {
			cb.log.Warnf("Unassigned ports are not supported: %s", port.Target)
			continue
		}

		portNumber, err := strconv.Atoi(port.Published)
		if err != nil {
			return err
		}

		ports = append(ports, map[string]interface{}{
			"port":          portNumber,
			"containerPort": int(port.Target),
			"protocol":      protocol,
		})
	}

	for _, port := range service.Expose {
		intPort, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("expected integer for port number: %s", err.Error())
		}
		ports = append(ports, map[string]interface{}{
			"port": intPort,
		})
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

	deployment := &latest.DeploymentConfig{
		Helm: &latest.HelmConfig{
			Values: values,
		},
	}

	if cb.config.Deployments == nil {
		cb.config.Deployments = map[string]*latest.DeploymentConfig{}
	}

	deploymentName := formatName(service.Name)
	cb.config.Deployments[deploymentName] = deployment

	return nil
}

func containerConfig(service composetypes.ServiceConfig, volumeMounts []interface{}) (map[string]interface{}, error) {
	container := map[string]interface{}{
		"name":  containerName(service),
		"image": resolveImage(service),
	}

	if len(service.Command) > 0 {
		container["args"] = shellCommandToSlice(service.Command)
	}

	if service.Build == nil && len(service.Entrypoint) > 0 {
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

func containerName(service composetypes.ServiceConfig) string {
	if service.ContainerName != "" {
		return formatName(service.ContainerName)
	}
	return fmt.Sprintf("%s-container", formatName(service.Name))
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

func shellCommandToSlice(command composetypes.ShellCommand) []interface{} {
	var slice []interface{}
	for _, item := range command {
		slice = append(slice, item)
	}
	return slice
}
