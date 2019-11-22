package kubectl

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/kubectl/walk"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"

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
}

// New creates a new deploy config for kubectl
func New(config *latest.Config, kubeClient kubectl.Client, deployConfig *latest.DeploymentConfig, log log.Logger) (deployer.Interface, error) {
	if deployConfig.Kubectl == nil {
		return nil, errors.New("Error creating kubectl deploy config: kubectl is nil")
	}
	if deployConfig.Kubectl.Manifests == nil {
		return nil, errors.New("No manifests defined for kubectl deploy")
	}

	namespace := kubeClient.Namespace()
	if deployConfig.Namespace != "" {
		namespace = deployConfig.Namespace
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
	}, nil
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

		cmd := exec.Command(d.CmdPath, args...)

		cmd.Stdin = stringReader
		cmd.Stdout = d.Log
		cmd.Stderr = d.Log

		err = cmd.Run()
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

			cmd := exec.Command(d.CmdPath, args...)

			cmd.Stdin = stringReader
			cmd.Stdout = d.Log
			cmd.Stderr = d.Log

			err = cmd.Run()
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
				shouldRedeploy = replaceManifest(manifestYaml, cache, d.config.Images, builtImages) || shouldRedeploy
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

	args = append(args, "--dry-run", "--output", "yaml", "--validate=false")

	if d.DeploymentConfig.Kubectl.Kustomize != nil && *d.DeploymentConfig.Kubectl.Kustomize == true {
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

func replaceManifest(manifest map[interface{}]interface{}, cache *generated.CacheConfig, imagesConf map[string]*latest.ImageConfig, builtImages map[string]string) bool {
	shouldRedeploy := false

	match := func(path, key, value string) bool {
		if key == "image" {
			image, err := registry.GetStrippedDockerImageName(value)
			if err != nil {
				return false
			}

			// Search for image name
			for _, imageCache := range cache.Images {
				found := false
				for _, imageConfig := range imagesConf {
					if imageConfig.Image == image {
						found = true
						break
					}
				}

				if found && imageCache.ImageName == image && imageCache.Tag != "" {
					if builtImages != nil {
						if _, ok := builtImages[image]; ok {
							shouldRedeploy = true
						}
					}

					return true
				}
			}
		}

		return false
	}

	replace := func(path, value string) (interface{}, error) {
		image, err := registry.GetStrippedDockerImageName(value)
		if err != nil {
			return false, nil
		}

		// Search for image name
		for _, imageCache := range cache.Images {
			if imageCache.ImageName == image {
				return image + ":" + imageCache.Tag, nil
			}
		}

		return value, nil
	}

	// We ignore the error here because the replace function can never throw an error
	_ = walk.Walk(manifest, match, replace)

	return shouldRedeploy
}
