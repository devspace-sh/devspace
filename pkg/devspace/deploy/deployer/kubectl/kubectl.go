package kubectl

import (
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// DeployConfig holds the necessary information for kubectl deployment
type DeployConfig struct {
	KubeClient kubectl.Client // This is not used yet, however the plan is to use it instead of calling kubectl via cmd
	Name       string
	CmdPath    string
	Context    string
	Namespace  string
	Manifests  []string

	DeploymentConfig *latest.DeploymentConfig
	Log              log.Logger

	config *latest.Config

	commandExecuter commandExecuter
}

// New creates a new deploy config for kubectl
func New(config *latest.Config, kubeClient kubectl.Client, deployConfig *latest.DeploymentConfig, log log.Logger) (deployer.Interface, error) {
	if deployConfig.Kubectl == nil {
		return nil, errors.New("Error creating kubectl deploy config: kubectl is nil")
	}
	if deployConfig.Kubectl.Manifests == nil {
		return nil, errors.New("No manifests defined for kubectl deploy")
	}

	cmdPath := "kubectl"
	if deployConfig.Kubectl.CmdPath != "" {
		cmdPath = deployConfig.Kubectl.CmdPath
	}

	manifests := []string{}
	for _, ptrManifest := range deployConfig.Kubectl.Manifests {
		manifest := strings.Replace(ptrManifest, "*", "", -1)
		if deployConfig.Kubectl.Kustomize != nil && *deployConfig.Kubectl.Kustomize == true {
			manifest = strings.TrimSuffix(manifest, "kustomization.yaml")
		}

		manifests = append(manifests, manifest)
	}

	if kubeClient == nil {
		return &DeployConfig{
			Name:       deployConfig.Name,
			KubeClient: kubeClient,
			CmdPath:    cmdPath,
			Manifests:  manifests,

			DeploymentConfig: deployConfig,
			config:           config,
			Log:              log,

			commandExecuter: &executer{},
		}, nil
	}

	namespace := kubeClient.Namespace()
	if deployConfig.Namespace != "" {
		namespace = deployConfig.Namespace
	}

	return &DeployConfig{
		Name:       deployConfig.Name,
		KubeClient: kubeClient,
		CmdPath:    cmdPath,
		Context:    kubeClient.CurrentContext(),
		Namespace:  namespace,
		Manifests:  manifests,

		DeploymentConfig: deployConfig,
		config:           config,
		Log:              log,

		commandExecuter: &executer{},
	}, nil
}

// Render writes the generated manifests to the out stream
func (d *DeployConfig) Render(cache *generated.CacheConfig, builtImages map[string]string, out io.Writer) error {
	for _, manifest := range d.Manifests {
		_, replacedManifest, err := d.getReplacedManifest(manifest, cache, builtImages)
		if err != nil {
			return errors.Errorf("%v\nPlease make sure `kubectl apply` does work locally with manifest `%s`", err, manifest)
		}

		out.Write([]byte(replacedManifest))
		out.Write([]byte("\n---\n"))
	}

	return nil
}

// Status prints the status of all matched manifests from kubernetes
func (d *DeployConfig) Status() (*deployer.StatusResult, error) {
	// TODO: parse kubectl get output into the required string array
	manifests := strings.Join(d.Manifests, ",")
	if len(manifests) > 20 {
		manifests = manifests[:20] + "..."
	}

	return &deployer.StatusResult{
		Name:   d.Name,
		Type:   "Manifests",
		Target: manifests,
		Status: "N/A",
	}, nil
}

// Delete deletes all matched manifests from kubernetes
func (d *DeployConfig) Delete(cache *generated.CacheConfig) error {
	d.Log.StartWait("Deleting manifests with kubectl")
	defer d.Log.StopWait()

	for _, manifest := range d.Manifests {
		_, replacedManifest, err := d.getReplacedManifest(manifest, cache, nil)
		if err != nil {
			return err
		}

		args := d.getCmdArgs("delete", "--ignore-not-found=true")
		stringReader := strings.NewReader(replacedManifest)

		cmd := d.commandExecuter.GetCommand(d.CmdPath, args)
		err = cmd.Run(d.Log, d.Log, stringReader)
		if err != nil {
			return err
		}
	}

	delete(cache.Deployments, d.DeploymentConfig.Name)
	return nil
}

// Deploy deploys all specified manifests via kubectl apply and adds to the specified image names the corresponding tags
func (d *DeployConfig) Deploy(cache *generated.CacheConfig, forceDeploy bool, builtImages map[string]string) (bool, error) {
	deployCache := cache.GetDeploymentCache(d.DeploymentConfig.Name)

	// Hash the manifests
	manifestsHash := ""
	for _, manifest := range d.Manifests {
		if strings.HasPrefix(manifest, "http://") || strings.HasPrefix(manifest, "https://") {
			manifestsHash += hash.String(manifest)
			continue
		}

		// Check if the chart directory has changed
		hash, err := hash.Directory(manifest)
		if err != nil {
			return false, errors.Errorf("Error hashing %s: %v", manifest, err)
		}

		manifestsHash += hash
	}

	// Hash the deployment config
	configStr, err := yaml.Marshal(d.DeploymentConfig)
	if err != nil {
		return false, errors.Wrap(err, "marshal deployment config")
	}

	deploymentConfigHash := hash.String(string(configStr))

	// We force the redeploy of kubectl deployments for now, because we don't know if they are already currently deployed or not,
	// so it is better to force deploy them, which usually takes almost no time and is better than taking the risk of skipping a needed deployment
	// forceDeploy = forceDeploy || deployCache.KubectlManifestsHash != manifestsHash || deployCache.DeploymentConfigHash != deploymentConfigHash
	forceDeploy = true

	d.Log.StartWait("Applying manifests with kubectl")
	defer d.Log.StopWait()

	wasDeployed := false

	for _, manifest := range d.Manifests {
		shouldRedeploy, replacedManifest, err := d.getReplacedManifest(manifest, cache, builtImages)
		if err != nil {
			return false, errors.Errorf("%v\nPlease make sure `kubectl apply` does work locally with manifest `%s`", err, manifest)
		}

		if shouldRedeploy || forceDeploy {
			stringReader := strings.NewReader(replacedManifest)
			args := d.getCmdArgs("apply", "--force")
			if d.DeploymentConfig.Kubectl.Flags != nil {
				for _, flag := range d.DeploymentConfig.Kubectl.Flags {
					args = append(args, flag)
				}
			}

			cmd := d.commandExecuter.GetCommand(d.CmdPath, args)
			err = cmd.Run(d.Log, d.Log, stringReader)
			if err != nil {
				return false, errors.Errorf("%v\nPlease make sure the command `kubectl apply` does work locally with manifest `%s`", err, manifest)
			}

			wasDeployed = true
		} else {
			d.Log.Infof("Skipping manifest %s", manifest)
		}
	}

	deployCache.KubectlManifestsHash = manifestsHash
	deployCache.DeploymentConfigHash = deploymentConfigHash

	return wasDeployed, nil
}

func (d *DeployConfig) getReplacedManifest(manifest string, cache *generated.CacheConfig, builtImages map[string]string) (bool, string, error) {
	manifestYamlBytes, err := d.dryRun(manifest)
	if err != nil {
		return false, "", err
	}

	// Split output into the yamls
	var (
		splitted         = regexp.MustCompile(`(^|\n)apiVersion`).Split(string(manifestYamlBytes), -1)
		replaceManifests = []string{}
		shouldRedeploy   = false
	)

	for _, resource := range splitted {
		if resource == "" {
			continue
		}

		// Parse yaml
		manifestYaml := map[interface{}]interface{}{}
		err = yaml.Unmarshal([]byte("apiVersion"+resource), &manifestYaml)
		if err != nil {
			return false, "", errors.Wrap(err, "unmarshal yaml")
		}

		if len(cache.Images) > 0 {
			if d.DeploymentConfig.Kubectl.ReplaceImageTags == nil || *d.DeploymentConfig.Kubectl.ReplaceImageTags == true {
				shouldRedeploy = util.ReplaceImageNames(manifestYaml, cache, d.config.Images, builtImages, map[string]bool{"image": true}) || shouldRedeploy
			}
		}

		replacedManifest, err := yaml.Marshal(manifestYaml)
		if err != nil {
			return false, "", errors.Wrap(err, "marshal yaml")
		}

		replaceManifests = append(replaceManifests, string(replacedManifest))
	}

	return shouldRedeploy, strings.Join(replaceManifests, "\n---\n"), nil
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

	args = append(args, "--dry-run", "--save-config", "--output", "yaml", "--validate=false")

	if d.DeploymentConfig.Kubectl.Kustomize != nil && *d.DeploymentConfig.Kubectl.Kustomize == true {
		args = append(args, "--kustomize")
	} else {
		args = append(args, "--filename")
	}

	args = append(args, manifest)

	// Execute command
	output, err := d.commandExecuter.RunCommand(d.CmdPath, args)
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			return nil, errors.New(string(exitError.Stderr))
		}

		return nil, err
	}

	return output, nil
}
