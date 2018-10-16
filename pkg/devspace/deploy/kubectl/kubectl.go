package kubectl

import (
	"errors"
	"os/exec"
	"strings"

	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
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
	// TODO: parse kubectl get output into the required string array
	return [][]string{}, nil
}

// Delete deletes all matched manifests from kubernetes
func (d *DeployConfig) Delete() error {
	manifests, err := loadManifests(d.Manifests, d.Log)
	if err != nil {
		return err
	}

	joinedManifests, err := joinManifests(manifests)
	if err != nil {
		return err
	}

	stringReader := strings.NewReader(joinedManifests)
	args := d.getCmdArgs("delete", "--ignore-not-found=true")

	cmd := exec.Command(d.CmdPath, args...)

	cmd.Stdin = stringReader
	cmd.Stdout = d.Log
	cmd.Stderr = d.Log

	return cmd.Run()
}

// Deploy deploys all specified manifests via kubectl apply and adds to the specified image names the corresponding tags
func (d *DeployConfig) Deploy(generatedConfig *generated.Config, forceDeploy bool) error {
	manifests, err := loadManifests(d.Manifests, d.Log)
	if err != nil {
		return err
	}

	for _, manifest := range manifests {
		replaceManifest(manifest, generatedConfig.ImageTags)
	}

	joinedManifests, err := joinManifests(manifests)
	if err != nil {
		return err
	}

	stringReader := strings.NewReader(joinedManifests)
	args := d.getCmdArgs("apply", "--force")

	cmd := exec.Command(d.CmdPath, args...)

	cmd.Stdin = stringReader
	cmd.Stdout = d.Log
	cmd.Stderr = d.Log

	return cmd.Run()
}

func (d *DeployConfig) getCmdArgs(method string, additionalArgs ...string) []string {
	args := []string{}

	if d.Namespace != "" {
		args = append(args, "-n", d.Namespace)
	}

	if d.Context != "" {
		args = append(args, "--context", d.Context)
	}

	args = append(args, method)

	if additionalArgs != nil {
		args = append(args, additionalArgs...)
	}

	args = append(args, "-f", "-")

	return args
}

func replaceManifest(manifest Manifest, tags map[string]string) {
	match := func(key, value string) bool {
		if key == "image" {
			if _, ok := tags[value]; ok {
				return true
			}
		}

		return false
	}

	replace := func(value string) string {
		return value + ":" + tags[value]
	}

	Walk(manifest, match, replace)
}
