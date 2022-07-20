package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mvdan.cc/sh/v3/expand"
	"os"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/legacy"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/stringutil"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/pkg/util/downloader"
	"github.com/loft-sh/devspace/pkg/util/downloader/commands"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer"
	"github.com/loft-sh/devspace/pkg/util/hash"
)

// DeployConfig holds the necessary information for kubectl deployment
type DeployConfig struct {
	Name        string
	CmdPath     string
	Context     string
	Namespace   string
	IsInCluster bool
	Manifests   []string

	DeploymentConfig *latest.DeploymentConfig
}

// New creates a new deploy config for kubectl
func New(ctx devspacecontext.Context, deployConfig *latest.DeploymentConfig) (deployer.Interface, error) {
	if deployConfig.Kubectl == nil {
		return nil, errors.New("error creating kubectl deploy config: kubectl is nil")
	} else if deployConfig.Kubectl.Manifests == nil {
		return nil, errors.New("no manifests defined for kubectl deploy")
	}

	// make sure kubectl exists
	var (
		err     error
		cmdPath string
	)
	if deployConfig.Kubectl.KubectlBinaryPath != "" {
		cmdPath = deployConfig.Kubectl.KubectlBinaryPath
	} else {
		cmdPath, err = downloader.NewDownloader(commands.NewKubectlCommand(), ctx.Log()).EnsureCommand(ctx.Context())
		if err != nil {
			return nil, err
		}
	}

	manifests := []string{}
	for _, ptrManifest := range deployConfig.Kubectl.Manifests {
		manifest := strings.ReplaceAll(ptrManifest, "*", "")
		if deployConfig.Kubectl.Kustomize != nil && *deployConfig.Kubectl.Kustomize {
			manifest = strings.TrimSuffix(manifest, "kustomization.yaml")
		}

		manifests = append(manifests, manifest)
	}

	if ctx.KubeClient() == nil {
		return &DeployConfig{
			Name:      deployConfig.Name,
			CmdPath:   cmdPath,
			Manifests: manifests,

			DeploymentConfig: deployConfig,
		}, nil
	}

	namespace := deployConfig.Namespace
	if namespace == "" {
		namespace = ctx.KubeClient().Namespace()
	}

	return &DeployConfig{
		Name:        deployConfig.Name,
		CmdPath:     cmdPath,
		Context:     ctx.KubeClient().CurrentContext(),
		Namespace:   namespace,
		Manifests:   manifests,
		IsInCluster: ctx.KubeClient().IsInCluster(),

		DeploymentConfig: deployConfig,
	}, nil
}

// Render writes the generated manifests to the out stream
func (d *DeployConfig) Render(ctx devspacecontext.Context, out io.Writer) error {
	for _, manifest := range d.Manifests {
		_, replacedManifest, _, err := d.getReplacedManifest(ctx, manifest)
		if err != nil {
			return errors.Errorf("%v\nPlease make sure `kubectl apply` does work locally with manifest `%s`", err, manifest)
		}

		_, _ = out.Write([]byte(replacedManifest))
		_, _ = out.Write([]byte("\n---\n"))
	}

	return nil
}

// Status prints the status of all matched manifests from kubernetes
func (d *DeployConfig) Status(ctx devspacecontext.Context) (*deployer.StatusResult, error) {
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

// Deploy deploys all specified manifests via kubectl apply and adds to the specified image names the corresponding tags
func (d *DeployConfig) Deploy(ctx devspacecontext.Context, _ bool) (bool, error) {
	deployCache, _ := ctx.Config().RemoteCache().GetDeployment(d.DeploymentConfig.Name)

	// Hash the manifests
	manifestsHash := ""
	for _, manifest := range d.Manifests {
		if strings.HasPrefix(manifest, "http://") || strings.HasPrefix(manifest, "https://") {
			manifestsHash += hash.String(manifest)
			continue
		}

		// Check if the chart directory has changed
		manifest = ctx.ResolvePath(manifest)
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
	forceDeploy := true

	ctx.Log().Info("Applying manifests with kubectl...")
	wasDeployed := false
	kubeObjects := []remotecache.KubectlObject{}
	writer := ctx.Log().Writer(logrus.InfoLevel, false)
	defer writer.Close()

	for _, manifest := range d.Manifests {
		shouldRedeploy, replacedManifest, parsedObjects, err := d.getReplacedManifest(ctx, manifest)
		if err != nil {
			return false, errors.Errorf("%v\nPlease make sure `kubectl apply` does work locally with manifest `%s`", err, manifest)
		}

		kubeObjects = append(kubeObjects, parsedObjects...)
		if shouldRedeploy || forceDeploy {
			args := d.getCmdArgs("apply", "--force")
			args = append(args, d.DeploymentConfig.Kubectl.ApplyArgs...)

			stdErrBuffer := &bytes.Buffer{}
			err = command.Command(ctx.Context(), ctx.WorkingDir(), ctx.Environ(), writer, io.MultiWriter(writer, stdErrBuffer), strings.NewReader(replacedManifest), d.CmdPath, args...)
			if err != nil {
				return false, errors.Errorf("%v %v\nPlease make sure the command `kubectl apply` does work locally with manifest `%s`", stdErrBuffer.String(), err, manifest)
			}

			wasDeployed = true
		} else {
			ctx.Log().Infof("Skipping manifest %s", manifest)
		}
	}

	deployCache.Kubectl = &remotecache.KubectlCache{
		Objects:       kubeObjects,
		ManifestsHash: manifestsHash,
	}
	deployCache.DeploymentConfigHash = deploymentConfigHash
	if rootName, ok := values.RootNameFrom(ctx.Context()); ok && !stringutil.Contains(deployCache.Projects, rootName) {
		deployCache.Projects = append(deployCache.Projects, rootName)
	}
	ctx.Config().RemoteCache().SetDeployment(d.DeploymentConfig.Name, deployCache)
	return wasDeployed, nil
}

func (d *DeployConfig) getReplacedManifest(ctx devspacecontext.Context, manifest string) (bool, string, []remotecache.KubectlObject, error) {
	objects, err := d.buildManifests(ctx, manifest)
	if err != nil {
		return false, "", nil, err
	}

	// Split output into the yamls
	var (
		replaceManifests = []string{}
		shouldRedeploy   = false
	)

	kubeObjects := []remotecache.KubectlObject{}
	for _, resource := range objects {
		if resource.Object == nil {
			continue
		}
		if resource.GetNamespace() == "" {
			resource.SetNamespace(d.Namespace)
		}

		kubeObjects = append(kubeObjects, remotecache.KubectlObject{
			APIVersion: resource.GetAPIVersion(),
			Kind:       resource.GetKind(),
			Name:       resource.GetName(),
			Namespace:  resource.GetNamespace(),
		})

		if d.DeploymentConfig.UpdateImageTags == nil || *d.DeploymentConfig.UpdateImageTags {
			redeploy, err := legacy.ReplaceImageNamesStringMap(resource.Object, ctx.Config(), ctx.Dependencies(), map[string]bool{"image": true})
			if err != nil {
				return false, "", nil, err
			} else if redeploy {
				shouldRedeploy = true
			}
		}

		replacedManifest, err := yaml.Marshal(resource)
		if err != nil {
			return false, "", nil, errors.Wrap(err, "marshal yaml")
		}

		replaceManifests = append(replaceManifests, string(replacedManifest))
	}

	return shouldRedeploy, strings.Join(replaceManifests, "\n---\n"), kubeObjects, nil
}

func (d *DeployConfig) getCmdArgs(method string, additionalArgs ...string) []string {
	args := []string{}
	if d.Context != "" && !d.IsInCluster {
		args = append(args, "--context", d.Context)
	}

	args = append(args, method)
	if additionalArgs != nil {
		args = append(args, additionalArgs...)
	}

	args = append(args, "-f", "-")
	return args
}

func (d *DeployConfig) buildManifests(ctx devspacecontext.Context, manifest string) ([]*unstructured.Unstructured, error) {
	// Check if we should use kustomize or kubectl
	kustomizePath := "kustomize"
	if d.DeploymentConfig.Kubectl.KustomizeBinaryPath != "" {
		kustomizePath = d.DeploymentConfig.Kubectl.KustomizeBinaryPath
	}

	if d.DeploymentConfig.Kubectl.Kustomize != nil && *d.DeploymentConfig.Kubectl.Kustomize && d.isKustomizeInstalled(ctx.Context(), ctx.WorkingDir(), kustomizePath) {
		return NewKustomizeBuilder(kustomizePath, d.DeploymentConfig, ctx.Log()).Build(ctx.Context(), ctx.Environ(), ctx.WorkingDir(), manifest)
	}

	raw, err := ctx.KubeClient().KubeConfigLoader().LoadConfig().RawConfig()
	if err != nil {
		return nil, fmt.Errorf("get raw config")
	}
	copied := raw.DeepCopy()
	for key := range copied.Contexts {
		copied.Contexts[key].Namespace = d.Namespace
	}

	// Build with kubectl
	return NewKubectlBuilder(d.CmdPath, d.DeploymentConfig, *copied).Build(ctx.Context(), ctx.Environ(), ctx.WorkingDir(), manifest)
}

func (d *DeployConfig) isKustomizeInstalled(ctx context.Context, dir, path string) bool {
	err := command.Command(ctx, dir, expand.ListEnviron(os.Environ()...), nil, nil, nil, path, "version")
	return err == nil
}
