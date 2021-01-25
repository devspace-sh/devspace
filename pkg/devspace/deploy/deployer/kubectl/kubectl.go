package kubectl

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/helm/downloader"
	"github.com/mitchellh/go-homedir"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"

	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/util"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/log"
)

var (
	kubectlVersion  = "v1.20.0"
	kubectlDownload = "https://storage.googleapis.com/kubernetes-release/release/" + kubectlVersion + "/bin/" + runtime.GOOS + "/amd64/kubectl"
)

// DeployConfig holds the necessary information for kubectl deployment
type DeployConfig struct {
	KubeClient  kubectl.Client // This is not used yet, however the plan is to use it instead of calling kubectl via cmd
	Name        string
	CmdPath     string
	Context     string
	Namespace   string
	IsInCluster bool
	Manifests   []string

	DeploymentConfig *latest.DeploymentConfig
	Log              log.Logger

	config *latest.Config

	commandExecuter commandExecuter
}

// New creates a new deploy config for kubectl
func New(config *latest.Config, kubeClient kubectl.Client, deployConfig *latest.DeploymentConfig, log log.Logger) (deployer.Interface, error) {
	if deployConfig.Kubectl == nil {
		return nil, errors.New("error creating kubectl deploy config: kubectl is nil")
	} else if deployConfig.Kubectl.Manifests == nil {
		return nil, errors.New("no manifests defined for kubectl deploy")
	}

	// make sure kubectl exists
	var (
		executer       = &executer{}
		isValidKubectl = func(command string) (bool, error) {
			return isValidKubectl(command, executer)
		}
		cmdPath = ""
	)
	if deployConfig.Kubectl.CmdPath != "" {
		cmdPath = deployConfig.Kubectl.CmdPath
	} else {
		home, err := homedir.Dir()
		if err != nil {
			return nil, err
		}

		installPath := filepath.Join(home, constants.DefaultHomeDevSpaceFolder, "bin", "kubectl")
		url := kubectlDownload
		if runtime.GOOS == "windows" {
			url += ".exe"
			installPath += ".exe"
		}

		cmdPath, err = downloader.NewDownloader(installKubectl, isValidKubectl, log).EnsureCLI("kubectl", installPath, url)
		if err != nil {
			return nil, err
		}
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

			commandExecuter: executer,
		}, nil
	}

	namespace := kubeClient.Namespace()
	if deployConfig.Namespace != "" {
		namespace = deployConfig.Namespace
	}

	return &DeployConfig{
		Name:        deployConfig.Name,
		KubeClient:  kubeClient,
		CmdPath:     cmdPath,
		Context:     kubeClient.CurrentContext(),
		Namespace:   namespace,
		Manifests:   manifests,
		IsInCluster: kubeClient.IsInCluster(),

		DeploymentConfig: deployConfig,
		config:           config,
		Log:              log,

		commandExecuter: executer,
	}, nil
}

func isValidKubectl(command string, executer *executer) (bool, error) {
	out, err := executer.RunCommand(command, []string{"version", "--client"})
	if err != nil {
		return false, nil
	}

	return strings.Index(string(out), `Client Version`) != -1, nil
}

func installKubectl(downloadedFile, installPath, installFromURL string) error {
	return copy.Copy(downloadedFile, installPath)
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
		args = append(args, d.DeploymentConfig.Kubectl.DeleteArgs...)

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
			args = append(args, d.DeploymentConfig.Kubectl.ApplyArgs...)

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
	objects, err := d.buildManifests(manifest)
	if err != nil {
		return false, "", err
	}

	// Split output into the yamls
	var (
		replaceManifests = []string{}
		shouldRedeploy   = false
	)

	for _, resource := range objects {
		if resource.Object == nil {
			continue
		}

		if len(cache.Images) > 0 {
			if d.DeploymentConfig.Kubectl.ReplaceImageTags == nil || *d.DeploymentConfig.Kubectl.ReplaceImageTags == true {
				shouldRedeploy = util.ReplaceImageNamesStringMap(resource.Object, cache, d.config.Images, builtImages, map[string]bool{"image": true}) || shouldRedeploy
			}
		}

		replacedManifest, err := yaml.Marshal(resource)
		if err != nil {
			return false, "", errors.Wrap(err, "marshal yaml")
		}

		replaceManifests = append(replaceManifests, string(replacedManifest))
	}

	return shouldRedeploy, strings.Join(replaceManifests, "\n---\n"), nil
}

func (d *DeployConfig) getCmdArgs(method string, additionalArgs ...string) []string {
	args := []string{}
	if d.Context != "" && d.IsInCluster == false {
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

func (d *DeployConfig) buildManifests(manifest string) ([]*unstructured.Unstructured, error) {
	// Check if we should use kustomize or kubectl
	if d.DeploymentConfig.Kubectl.Kustomize != nil && *d.DeploymentConfig.Kubectl.Kustomize == true && d.isKustomizeInstalled("kustomize") {
		return NewKustomizeBuilder("kustomize", d.DeploymentConfig, d.Log).Build(manifest, d.commandExecuter.RunCommand)
	}

	// Build with kubectl
	return NewKubectlBuilder(d.CmdPath, d.DeploymentConfig, d.Context, d.Namespace, d.IsInCluster).Build(manifest, d.commandExecuter.RunCommand)
}

func (d *DeployConfig) isKustomizeInstalled(path string) bool {
	_, err := d.commandExecuter.RunCommand(path, []string{"version"})
	if err != nil {
		return false
	}

	return true
}
