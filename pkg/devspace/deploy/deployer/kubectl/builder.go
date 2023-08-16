package kubectl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/env"
	"github.com/loft-sh/devspace/pkg/util/constraint"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/utils/pkg/command"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/cmd/version"
	"mvdan.cc/sh/v3/expand"
	jsonyaml "sigs.k8s.io/yaml"
)

// Builder is the manifest builder interface
type Builder interface {
	Build(ctx context.Context, environ expand.Environ, dir, manifest string) ([]*unstructured.Unstructured, error)
}

type kustomizeBuilder struct {
	path   string
	config *latest.DeploymentConfig
	log    log.Logger
}

func NewKustomizeBuilder(path string, config *latest.DeploymentConfig, log log.Logger) Builder {
	return &kustomizeBuilder{
		path:   path,
		config: config,
		log:    log,
	}
}

func (k *kustomizeBuilder) Build(ctx context.Context, environ expand.Environ, dir, manifest string) ([]*unstructured.Unstructured, error) {
	args := []string{"build", manifest}
	args = append(args, k.config.Kubectl.KustomizeArgs...)

	// Execute command
	k.log.Infof("Render manifests with 'kustomize %s'", strings.Join(args, " "))
	output, err := command.Output(ctx, dir, environ, k.path, args...)
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			return nil, errors.New(string(exitError.Stderr))
		}

		return nil, err
	}

	return stringToUnstructuredArray(string(output))
}

type kubectlBuilder struct {
	path       string
	config     *latest.DeploymentConfig
	kubeConfig clientcmdapi.Config
}

// NewKubectlBuilder creates a new kubectl manifest builder
func NewKubectlBuilder(path string, config *latest.DeploymentConfig, kubeConfig clientcmdapi.Config) Builder {
	return &kubectlBuilder{
		path:       path,
		config:     config,
		kubeConfig: kubeConfig,
	}
}

// this function is called in Build function
// to decide the --dry-run value
var useOldDryRun = func(ctx context.Context, environ expand.Environ, dir, path string) (bool, error) {
	// compare kubectl version for --dry-run flag value
	out, err := command.Output(ctx, dir, environ, path, "version", "--client", "--output=json")
	if err != nil {
		return false, err
	}

	kubectlVersion := &version.Version{}
	err = json.Unmarshal(out, kubectlVersion)
	if err != nil {
		return false, err
	}

	v1, err := constraint.NewVersion(strings.TrimPrefix(kubectlVersion.ClientVersion.GitVersion, "v"))
	if err != nil {
		return false, nil
	}

	v2, err := constraint.NewVersion("1.18.0")
	if err != nil {
		return false, err
	}

	if v1.LessThan(v2) {
		return true, nil
	}

	return false, nil
}

func (k *kubectlBuilder) Build(ctx context.Context, environ expand.Environ, dir, manifest string) ([]*unstructured.Unstructured, error) {
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempFile.Name())

	data, err := clientcmd.Write(k.kubeConfig)
	if err != nil {
		return nil, err
	}

	_, err = tempFile.Write(data)
	if err != nil {
		return nil, err
	}
	_ = tempFile.Close()

	args := []string{"create"}

	// decides which --dry-run value is to be used
	uodr, err := useOldDryRun(ctx, environ, dir, k.path)
	if err != nil {
		return nil, err
	}

	if uodr {
		args = append(args, "--dry-run", "--output", "yaml", "--validate=false")
	} else {
		args = append(args, "--dry-run=client", "--output", "yaml", "--validate=false")
	}

	if k.config.Kubectl.Kustomize != nil && *k.config.Kubectl.Kustomize {
		args = append(args, "--kustomize", manifest)
	} else {
		args = append(args, "--filename", manifest)
	}

	// Add extra args
	args = append(args, k.config.Kubectl.CreateArgs...)

	// Execute command
	output, err := command.Output(ctx, dir, env.NewVariableEnvProvider(environ, map[string]string{
		"KUBECONFIG": tempFile.Name(),
	}), k.path, args...)
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			return nil, errors.New(string(exitError.Stderr))
		}

		return nil, err
	}

	return stringToUnstructuredArray(string(output))
}

var diffSeparator = regexp.MustCompile(`\n---`)

// stringToUnstructuredArray splits a YAML file into unstructured objects. Returns a list of all unstructured objects
func stringToUnstructuredArray(out string) ([]*unstructured.Unstructured, error) {
	parts := diffSeparator.Split(out, -1)
	var objs []*unstructured.Unstructured
	var firstErr error
	for _, part := range parts {
		var objMap map[string]interface{}
		err := jsonyaml.Unmarshal([]byte(part), &objMap)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to unmarshal manifest: %v", err)
			}
			continue
		}
		if len(objMap) == 0 {
			// handles case where theres no content between `---`
			continue
		}
		var obj unstructured.Unstructured
		err = jsonyaml.Unmarshal([]byte(part), &obj)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to unmarshal manifest: %v", err)
			}
			continue
		}
		objs = append(objs, &obj)
	}
	return objs, firstErr
}
