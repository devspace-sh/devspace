package compose

import (
	"fmt"

	"github.com/loft-sh/devspace/pkg/util/log"

	composetypes "github.com/compose-spec/compose-go/types"
)

func GetServiceSyncPaths(
	project *composetypes.Project,
	service composetypes.ServiceConfig,
) []string {
	syncPaths := []string{}
	for _, volumeMount := range service.Volumes {
		isProvisionedVolume := false
		for _, volume := range project.Volumes {
			if volumeMount.Source == volume.Name {
				isProvisionedVolume = true
			}
		}

		if !isProvisionedVolume {
			syncPaths = append(syncPaths, volumeMount.Target)
		}
	}
	return syncPaths
}

func volumesConfig(
	service composetypes.ServiceConfig,
	composeVolumes map[string]composetypes.VolumeConfig,
	log log.Logger,
) (volumes []interface{}, volumeMounts []interface{}, bindVolumeMounts []interface{}) {
	for _, secret := range service.Secrets {
		volume := createSecretVolume(secret)
		volumes = append(volumes, volume)

		volumeMount := createSecretVolumeMount(secret)
		volumeMounts = append(volumeMounts, volumeMount)
	}

	var volumeVolumes []composetypes.ServiceVolumeConfig
	var bindVolumes []composetypes.ServiceVolumeConfig
	var tmpfsVolumes []composetypes.ServiceVolumeConfig
	for _, serviceVolume := range service.Volumes {
		switch serviceVolume.Type {
		case composetypes.VolumeTypeBind:
			bindVolumes = append(bindVolumes, serviceVolume)
		case composetypes.VolumeTypeTmpfs:
			tmpfsVolumes = append(tmpfsVolumes, serviceVolume)
		case composetypes.VolumeTypeVolume:
			volumeVolumes = append(volumeVolumes, serviceVolume)
		default:
			log.Warnf("%s volumes are not supported", serviceVolume.Type)
		}
	}

	volumeMap := map[string]interface{}{}
	for idx, volumeVolume := range volumeVolumes {
		volumeName := volumeName(service, volumeVolume, idx+1)
		_, ok := volumeMap[volumeName]
		if !ok {
			volume := createVolume(volumeName, DefaultVolumeSize)
			volumes = append(volumes, volume)
			volumeMap[volumeName] = volume
		}

		volumeMount := createSharedVolumeMount(volumeName, volumeVolume)
		volumeMounts = append(volumeMounts, volumeMount)
	}

	for _, tmpfsVolume := range tmpfsVolumes {
		volumeName := volumeName(service, tmpfsVolume, len(volumes))
		volume := createEmptyDirVolume(volumeName, tmpfsVolume)
		volumes = append(volumes, volume)

		volumeMount := createServiceVolumeMount(volumeName, tmpfsVolume)
		volumeMounts = append(volumeMounts, volumeMount)
	}

	for idx, bindVolume := range bindVolumes {
		volumeName := fmt.Sprintf("volume-%d", idx+1)
		volume := createEmptyDirVolume(volumeName, bindVolume)
		volumes = append(volumes, volume)

		volumeMount := createServiceVolumeMount(volumeName, bindVolume)
		volumeMounts = append(volumeMounts, volumeMount)
	}

	return volumes, volumeMounts, bindVolumeMounts
}

func createEmptyDirVolume(volumeName string, volume composetypes.ServiceVolumeConfig) interface{} {
	emptyDir := map[string]interface{}{}
	if volume.Tmpfs != nil {
		emptyDir["sizeLimit"] = fmt.Sprintf("%d", volume.Tmpfs.Size)
	}
	return map[string]interface{}{
		"name":     volumeName,
		"emptyDir": emptyDir,
	}
}

func createSecretVolume(secret composetypes.ServiceSecretConfig) interface{} {
	return map[string]interface{}{
		"name": secret.Source,
		"secret": map[string]interface{}{
			"secretName": secret.Source,
		},
	}
}

func createSecretVolumeMount(secret composetypes.ServiceSecretConfig) interface{} {
	target := secret.Source
	if secret.Target != "" {
		target = secret.Target
	}
	return map[string]interface{}{
		"containerPath": fmt.Sprintf("/run/secrets/%s", target),
		"volume": map[string]interface{}{
			"name":     secret.Source,
			"subPath":  target,
			"readOnly": true,
		},
	}
}

func createSharedVolumeMount(volumeName string, volume composetypes.ServiceVolumeConfig) interface{} {
	volumeConfig := map[string]interface{}{
		"name":   volumeName,
		"shared": true,
	}

	if volume.ReadOnly {
		volumeConfig["readOnly"] = true
	}

	return map[string]interface{}{
		"containerPath": volume.Target,
		"volume":        volumeConfig,
	}
}

func createServiceVolumeMount(volumeName string, volume composetypes.ServiceVolumeConfig) interface{} {
	readonly := volume.ReadOnly
	if volume.Source != "" {
		readonly = false
	}
	return map[string]interface{}{
		"containerPath": volume.Target,
		"volume": map[string]interface{}{
			"name":     volumeName,
			"readOnly": readonly,
		},
	}
}

func createVolume(name string, size string) interface{} {
	return map[string]interface{}{
		"name": name,
		"size": size,
	}
}

func volumeName(service composetypes.ServiceConfig, volume composetypes.ServiceVolumeConfig, idx int) string {
	volumeName := volume.Source
	if volumeName == "" {
		volumeName = fmt.Sprintf("%s-%d", formatName(service.Name), idx)
	}
	return volumeName
}
