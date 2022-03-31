package compose

import (
	"io/ioutil"
	"os"
	"path/filepath"

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

type ComposeManager interface {
	Load(log log.Logger) error
	Config(path string) *latest.Config
	Configs() map[string]*latest.Config
	Save() error
}

type composeManager struct {
	composePath string
	configs     map[string]*latest.Config
}

func NewComposeManager(composePath string) ComposeManager {
	return &composeManager{
		composePath: composePath,
		configs:     map[string]*latest.Config{},
	}
}

func (cm *composeManager) Load(log log.Logger) error {
	composeFile, err := ioutil.ReadFile(cm.composePath)
	if err != nil {
		return err
	}

	dockerCompose, err := composeloader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{
				Content: composeFile,
			},
		},
	})
	if err != nil {
		return err
	}

	dependentsMap, err := calculateDependentsMap(dockerCompose)
	if err != nil {
		return err
	}

	builders := map[string]ConfigBuilder{}
	err = dockerCompose.WithServices(nil, func(service composetypes.ServiceConfig) error {
		configKey := constants.DefaultConfigPath
		configName := "docker-compose"

		isDependency := dependentsMap[service.Name] != nil
		if isDependency {
			configKey = "devspace-" + service.Name + ".yaml"
			configName = service.Name
		}

		builder := builders[configKey]
		if builder == nil {
			workingDir := filepath.Dir(cm.composePath)
			builder = NewConfigBuilder(workingDir, log)
			builders[configKey] = builder
		}

		builder.SetName(configName)

		err := builder.AddImage(service)
		if err != nil {
			return err
		}

		err = builder.AddDeployment(*dockerCompose, service)
		if err != nil {
			return err
		}

		err = builder.AddDev(service)
		if err != nil {
			return err
		}

		err = builder.AddSecret(*dockerCompose, service)
		if err != nil {
			return err
		}

		err = builder.AddDependencies(*dockerCompose, service)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	for path, builder := range builders {
		cm.configs[path] = builder.Config()
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

		err = ioutil.WriteFile(path, configYaml, os.ModePerm)
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
