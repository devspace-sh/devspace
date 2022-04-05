package compose

import (
	"regexp"

	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
)

type ConfigBuilder interface {
	AddDependencies(dependency *composetypes.Project, service composetypes.ServiceConfig) error
	AddDeployment(dockerCompose *composetypes.Project, service composetypes.ServiceConfig) error
	AddDev(service composetypes.ServiceConfig) error
	AddImage(dockerCompose *composetypes.Project, service composetypes.ServiceConfig) error
	AddSecret(dockerCompose *composetypes.Project, service composetypes.ServiceConfig) error
	Config() *latest.Config
	SetName(name string)
}

type configBuilder struct {
	config     *latest.Config
	log        log.Logger
	workingDir string
}

func NewConfigBuilder(workingDir string, log log.Logger) ConfigBuilder {
	return &configBuilder{
		config:     latest.New().(*latest.Config),
		log:        log,
		workingDir: workingDir,
	}
}

func (cb *configBuilder) Config() *latest.Config {
	return cb.config
}

func (cb *configBuilder) SetName(name string) {
	cb.config.Name = name
}

func formatName(name string) string {
	return regexp.MustCompile(`[\._]`).ReplaceAllString(name, "-")
}

func labelSelector(serviceName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": serviceName,
	}
}
