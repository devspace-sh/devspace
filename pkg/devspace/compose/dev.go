package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func (cb *configBuilder) AddDev(service composetypes.ServiceConfig) error {
	var dev *latest.DevPod

	devPorts := []*latest.PortMapping{}
	for _, port := range service.Ports {

		portMapping := &latest.PortMapping{}

		if port.Published == "" {
			cb.log.Warnf("Unassigned port ranges are not supported: %s", port.Target)
			continue
		}

		portNumber, err := strconv.Atoi(port.Published)
		if err != nil {
			return err
		}

		if portNumber != int(port.Target) {
			portMapping.Port = fmt.Sprint(port.Published) + ":" + fmt.Sprint(port.Target)
		} else {
			portMapping.Port = fmt.Sprint(port.Published)
		}

		if port.HostIP != "" {
			portMapping.BindAddress = port.HostIP
		}

		devPorts = append(devPorts, portMapping)
	}

	for _, expose := range service.Expose {
		devPorts = append(devPorts, &latest.PortMapping{
			Port: expose,
		})
	}

	syncConfigs := []*latest.SyncConfig{}
	for _, volume := range service.Volumes {
		if volume.Type == composetypes.VolumeTypeBind {
			sync := &latest.SyncConfig{
				Path:           strings.Join([]string{resolveLocalPath(volume), volume.Target}, ":"),
				StartContainer: true,
			}

			_, err := os.Stat(filepath.Join(cb.workingDir, volume.Source, DockerIgnorePath))
			if err == nil {
				sync.ExcludeFile = DockerIgnorePath
			}

			syncConfigs = append(syncConfigs, sync)
		}
	}

	if len(devPorts) > 0 || len(syncConfigs) > 0 {
		dev = &latest.DevPod{
			LabelSelector: labelSelector(service.Name),
		}
	}

	if len(devPorts) > 0 {
		dev.Ports = devPorts
	}

	if len(syncConfigs) > 0 {
		dev.Sync = syncConfigs
		dev.Command = service.Entrypoint
	}

	if dev != nil {
		if cb.config.Dev == nil {
			cb.config.Dev = map[string]*latest.DevPod{}
		}

		devName := formatName(service.Name)
		cb.config.Dev[devName] = dev
	}

	return nil
}

func resolveLocalPath(volume composetypes.ServiceVolumeConfig) string {
	localSubPath := volume.Source

	if strings.HasPrefix(localSubPath, "~") {
		localSubPath = fmt.Sprintf(`${devspace.userHome}/%s`, strings.TrimLeft(localSubPath, "~/"))
	}
	return localSubPath
}
