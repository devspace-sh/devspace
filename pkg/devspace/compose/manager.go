package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	composeloader "github.com/compose-spec/compose-go/loader"
	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
	"gopkg.in/yaml.v3"
)

var (
	DockerComposePaths         = []string{"docker-compose.yaml", "docker-compose.yml"}
	DockerIgnorePath           = ".dockerignore"
	DefaultVolumeSize          = "5Gi"
	UploadVolumesContainerName = "upload-volumes"
)

func GetDockerComposePath() string {
	for _, composePath := range DockerComposePaths {
		_, err := os.Stat(composePath)
		if err == nil {
			return composePath
		}
	}
	return ""
}

func LoadDockerComposeProject(path string) (*composetypes.Project, error) {
	composeFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	project, err := composeloader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{
				Content: composeFile,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	// Expand service ports
	for idx, service := range project.Services {
		ports := []composetypes.ServicePortConfig{}
		for _, port := range service.Ports {
			expandedPorts, err := expandPublishedPortRange(port)
			if err != nil {
				return nil, err
			}
			ports = append(ports, expandedPorts...)
		}
		project.Services[idx].Ports = ports
	}

	return project, nil
}

type ComposeManager interface {
	Load(log log.Logger) error
	Configs() map[string]*latest.Config
	Save() error
}

type composeManager struct {
	configs map[string]*latest.Config
	project *composetypes.Project
}

func NewComposeManager(project *composetypes.Project) ComposeManager {
	return &composeManager{
		configs: map[string]*latest.Config{},
		project: project,
	}
}

func (cm *composeManager) Load(log log.Logger) error {
	dependentsMap, err := calculateDependentsMap(cm.project)
	if err != nil {
		return err
	}

	builders := map[string]ConfigBuilder{}
	err = cm.project.WithServices(nil, func(service composetypes.ServiceConfig) error {
		configName := "docker-compose"
		workingDir := cm.project.WorkingDir

		isDependency := dependentsMap[service.Name] != nil
		if isDependency {
			configName = service.Name
			if service.Build != nil && service.Build.Context != "" {
				workingDir = filepath.Join(workingDir, service.Build.Context)
			}
		}

		builder := builders[configName]
		if builder == nil {
			builder = NewConfigBuilder(workingDir, log)
			builders[configName] = builder
		}

		builder.SetName(configName)

		err := builder.AddImage(cm.project, service)
		if err != nil {
			return err
		}

		err = builder.AddDeployment(cm.project, service)
		if err != nil {
			return err
		}

		err = builder.AddDev(service)
		if err != nil {
			return err
		}

		err = builder.AddSecret(cm.project, service)
		if err != nil {
			return err
		}

		err = builder.AddDependencies(cm.project, service)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = cm.project.WithServices(nil, func(service composetypes.ServiceConfig) error {
		configName := "docker-compose"
		path := constants.DefaultConfigPath

		isDependency := dependentsMap[service.Name] != nil
		if isDependency {
			configName = service.Name

			path = "devspace-" + service.Name + ".yaml"
			if service.Build != nil && service.Build.Context != "" {
				path = filepath.Join(service.Build.Context, "devspace.yaml")
			}
		}

		builder := builders[configName]
		cm.configs[path] = builder.Config()

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (cm *composeManager) Config(path string) *latest.Config {
	return cm.configs[path]
}

func (cm *composeManager) Configs() map[string]*latest.Config {
	return cm.configs
}

func (cm *composeManager) Save() error {
	for path, config := range cm.configs {
		configYaml, err := yaml.Marshal(config)
		if err != nil {
			return err
		}

		err = os.WriteFile(path, configYaml, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
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

func expandPublishedPortRange(port composetypes.ServicePortConfig) ([]composetypes.ServicePortConfig, error) {
	if !strings.Contains(port.Published, "-") {
		return []composetypes.ServicePortConfig{port}, nil
	}

	publishedRange := strings.Split(port.Published, "-")
	if len(publishedRange) > 2 {
		return nil, fmt.Errorf("invalid port range")
	}

	begin, err := strconv.Atoi(publishedRange[0])
	if err != nil {
		return nil, fmt.Errorf("invalid port range %s: beginning value must be numeric", port.Published)
	}

	end, err := strconv.Atoi(publishedRange[1])
	if err != nil {
		return nil, fmt.Errorf("invalid port range %s: end value must be numeric", port.Published)
	}

	var portConfigs []composetypes.ServicePortConfig
	for i := begin; i <= end; i++ {
		portConfigs = append(portConfigs, composetypes.ServicePortConfig{
			HostIP:    port.HostIP,
			Protocol:  port.Protocol,
			Target:    port.Target,
			Published: strconv.Itoa(i),
			Mode:      "ingress",
		})
	}

	return portConfigs, nil
}
