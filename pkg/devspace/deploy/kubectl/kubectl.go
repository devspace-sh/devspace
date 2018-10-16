package kubectl

import (
	"errors"
	"path/filepath"

	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/yamlutil"
)

// DeployConfig holds the necessary information for kubectl deployment
type DeployConfig struct {
	KubeClient *kubernetes.Clientset // This is not used yet, however the plan is to use it instead of calling kubectl via cmd
	CmdPath    string
	Context    string
	Namespace  string
	Manifests  []string
	Log        log.Logger
}

// New creates a new deploy config for kubectl
func New(kubectl *kubernetes.Clientset, deployConfig *v1.DeploymentConfig, log log.Logger) (*DeployConfig, error) {
	if deployConfig.Kubectl == nil {
		return nil, errors.New("Error creating kubectl deploy config: kubectl is nil")
	}
	if deployConfig.Kubectl.Manifests == nil {
		return nil, errors.New("No manifests defined for kubectl deploy")
	}

	config := configutil.GetConfig()

	context := ""
	if config.Cluster != nil && config.Cluster.KubeContext != nil {
		context = *config.Cluster.KubeContext
	}

	namespace := ""
	if deployConfig.Namespace != nil {
		namespace = *deployConfig.Namespace
	}

	cmdPath := "kubectl"
	if deployConfig.Kubectl.CmdPath != nil {
		cmdPath = *deployConfig.Kubectl.CmdPath
	}

	manifests := []string{}
	for _, manifest := range *deployConfig.Kubectl.Manifests {
		manifests = append(manifests, *manifest)
	}

	return &DeployConfig{
		KubeClient: kubectl,
		CmdPath:    cmdPath,
		Context:    context,
		Namespace:  namespace,
		Manifests:  manifests,
		Log:        log,
	}, nil
}

// Status prints the status of all matched manifests from kubernetes
func (d *DeployConfig) Status() ([][]string, error) {
	return nil, nil
}

// Delete deletes all matched manifests from kubernetes
func (d *DeployConfig) Delete() error {
	return nil
}

// Deploy deploys all specified manifests via kubectl apply and adds to the specified image names the corresponding tags
func (d *DeployConfig) Deploy(generatedConfig *generated.Config, forceDeploy bool) error {
	return nil
}

func (d *DeployConfig) internalDeploy(images []string, tags map[string]string) error {
	for _, pattern := range d.Manifests {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}

		for _, file := range files {
			err = applyFile(d.Context, file, d.Namespace, images, tags)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func applyFile(context, file, namespace string, images []string, tags map[string]string) error {
	y := make(map[interface{}]interface{})
	yamlutil.ReadYamlFromFile(file, y)

	match := func(key, value string) bool {
		return false
	}

	replace := func(value string) string {
		return ""
	}

	Walk(y, match, replace)

	//changedManifest, err := yaml.Marshal(y)
	//if err != nil {
	//	return err
	//}

	return nil
}
