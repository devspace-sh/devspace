package kubectl

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/kubectl/walk"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// DeployConfig holds the necessary information for kubectl deployment
type DeployConfig struct {
	KubeClient *kubernetes.Clientset // This is not used yet, however the plan is to use it instead of calling kubectl via cmd
	Name       string
	CmdPath    string
	Context    string
	Namespace  string
	Manifests  []string

	Options *latest.KubectlConfig
	Log     log.Logger
}

// New creates a new deploy config for kubectl
func New(kubectl *kubernetes.Clientset, deployConfig *latest.DeploymentConfig, log log.Logger) (*DeployConfig, error) {
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

	namespace, err := configutil.GetDefaultNamespace(config)
	if err != nil {
		return nil, err
	}
	if deployConfig.Namespace != nil && *deployConfig.Namespace != "" {
		namespace = *deployConfig.Namespace
	}

	cmdPath := "kubectl"
	if deployConfig.Kubectl.CmdPath != nil {
		cmdPath = *deployConfig.Kubectl.CmdPath
	}

	manifests := []string{}
	for _, ptrManifest := range *deployConfig.Kubectl.Manifests {
		manifest := strings.ReplaceAll(*ptrManifest, "*", "")
		if deployConfig.Kubectl.Kustomize != nil && *deployConfig.Kubectl.Kustomize == true {
			manifest = strings.TrimSuffix(manifest, "kustomization.yaml")
		}

		manifests = append(manifests, manifest)
	}

	return &DeployConfig{
		Name:       *deployConfig.Name,
		KubeClient: kubectl,
		CmdPath:    cmdPath,
		Context:    context,
		Namespace:  namespace,
		Manifests:  manifests,
		Options:    deployConfig.Kubectl,
		Log:        log,
	}, nil
}

// Status prints the status of all matched manifests from kubernetes
func (d *DeployConfig) Status() (*deploy.StatusResult, error) {
	// TODO: parse kubectl get output into the required string array
	manifests := strings.Join(d.Manifests, ",")
	if len(manifests) > 20 {
		manifests = manifests[:20] + "..."
	}

	return &deploy.StatusResult{
		Name:   d.Name,
		Type:   "Manifests",
		Target: manifests,
		Status: "N/A",
	}, nil
}

// Delete deletes all matched manifests from kubernetes
func (d *DeployConfig) Delete() error {
	d.Log.StartWait("Deleting manifests with kubectl")
	defer d.Log.StopWait()

	for _, manifest := range d.Manifests {
		replacedManifest, err := d.getReplacedManifest(manifest, nil)
		if err != nil {
			return err
		}

		args := d.getCmdArgs("delete", "--ignore-not-found=true")
		stringReader := strings.NewReader(replacedManifest)

		cmd := exec.Command(d.CmdPath, args...)

		cmd.Stdin = stringReader
		cmd.Stdout = d.Log
		cmd.Stderr = d.Log

		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

// Deploy deploys all specified manifests via kubectl apply and adds to the specified image names the corresponding tags
func (d *DeployConfig) Deploy(generatedConfig *generated.Config, isDev, forceDeploy bool) error {
	d.Log.StartWait("Applying manifests with kubectl")
	defer d.Log.StopWait()

	activeConfig := generatedConfig.GetActive().Deploy
	if isDev {
		activeConfig = generatedConfig.GetActive().Dev
	}

	for _, manifest := range d.Manifests {
		replacedManifest, err := d.getReplacedManifest(manifest, activeConfig.ImageTags)
		if err != nil {
			return err
		}

		stringReader := strings.NewReader(replacedManifest)
		args := d.getCmdArgs("apply", "--force")
		if d.Options.Flags != nil {
			for _, flag := range *d.Options.Flags {
				args = append(args, *flag)
			}
		}

		cmd := exec.Command(d.CmdPath, args...)

		cmd.Stdin = stringReader
		cmd.Stdout = d.Log
		cmd.Stderr = d.Log

		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DeployConfig) getReplacedManifest(manifest string, imageTags map[string]string) (string, error) {
	manifestYamlBytes, err := d.dryRun(manifest)
	if err != nil {
		return "", err
	}

	// Split output into the yamls
	splitted := regexp.MustCompile(`(^|\n)apiVersion`).Split(string(manifestYamlBytes), -1)
	replaceManifests := []string{}

	for _, resource := range splitted {
		if resource == "" {
			continue
		}

		// Parse yaml
		manifestYaml := map[interface{}]interface{}{}
		err = yaml.Unmarshal([]byte("apiVersion"+resource), &manifestYaml)
		if err != nil {
			return "", errors.Wrap(err, "unmarshal yaml")
		}

		if len(imageTags) > 0 {
			replaceManifest(manifestYaml, imageTags)
		}

		replacedManifest, err := yaml.Marshal(manifestYaml)
		if err != nil {
			return "", errors.Wrap(err, "marshal yaml")
		}

		replaceManifests = append(replaceManifests, string(replacedManifest))
	}

	return strings.Join(replaceManifests, "\n---\n"), nil
}

func (d *DeployConfig) getCmdArgs(method string, additionalArgs ...string) []string {
	args := []string{}

	if d.Context != "" {
		args = append(args, "--context", d.Context)
	}
	if d.Namespace != "" {
		args = append(args, "--namespace", d.Namespace)
	}

	args = append(args, method)

	if additionalArgs != nil {
		args = append(args, additionalArgs...)
	}

	args = append(args, "-f", "-")

	return args
}

func (d *DeployConfig) dryRun(manifest string) ([]byte, error) {
	args := []string{"create"}

	if d.Context != "" {
		args = append(args, "--context", d.Context)
	}
	if d.Namespace != "" {
		args = append(args, "--namespace", d.Namespace)
	}

	args = append(args, "--dry-run", "--output", "yaml", "--validate=false")

	if d.Options.Kustomize != nil && *d.Options.Kustomize == true {
		args = append(args, "--kustomize")
	} else {
		args = append(args, "--filename")
	}

	args = append(args, manifest)

	// Execute command
	output, err := exec.Command(d.CmdPath, args...).Output()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			return nil, errors.New(string(exitError.Stderr))
		}

		return nil, err
	}

	return output, nil
}

func replaceManifest(manifest map[interface{}]interface{}, tags map[string]string) {
	match := func(path, key, value string) bool {
		if key == "image" {
			if _, ok := tags[value]; ok {
				return true
			}
		}

		return false
	}

	replace := func(path, value string) interface{} {
		return value + ":" + tags[value]
	}

	walk.Walk(manifest, match, replace)
}
