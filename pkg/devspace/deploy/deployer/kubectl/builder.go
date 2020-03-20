package kubectl

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os/exec"
	"regexp"
)

// Builder is the manifest builder interface
type Builder interface {
	Build(manifest string) ([]*unstructured.Unstructured, error)
}

type kustomizeBuilder struct {
	path string
	config *latest.DeploymentConfig
	cmd commandExecuter
}

func NewKustomizeBuilder(path string, config *latest.DeploymentConfig) Builder {
	return &kustomizeBuilder{
		path: path,
		config: config,
		cmd: &executer{},
	}
}

func (k *kustomizeBuilder) Build(manifest string) ([]*unstructured.Unstructured, error) {
	args := []string{"build", manifest}
	args = append(args, k.config.Kubectl.KustomizeArgs...)

	// Execute command
	output, err := k.cmd.RunCommand(k.path, args)
	if err != nil {
		_, ok := err.(*exec.ExitError)
		if ok {
			return nil, errors.New(string(output))
		}

		return nil, err
	}

	return stringToUnstructuredArray(string(output))
}

type kubectlBuilder struct {
	path string
	config *latest.DeploymentConfig
	context string
	namespace string
	cmd commandExecuter
}

// NewKubectlBuilder creates a new kubectl manifest builder
func NewKubectlBuilder(path string, config *latest.DeploymentConfig, context, namespace string) Builder {
	return &kubectlBuilder{
		path: path,
		config: config,
		context: context,
		namespace: namespace,
		cmd: &executer{},
	}
}

func (k *kubectlBuilder) Build(manifest string) ([]*unstructured.Unstructured, error) {
	args := []string{"create"}
	if k.context != "" {
		args = append(args, "--context", k.context)
	}
	if k.namespace != "" {
		args = append(args, "--namespace", k.namespace)
	}

	args = append(args, "--dry-run", "--output", "yaml", "--validate=false")
	if k.config.Kubectl.Kustomize != nil && *k.config.Kubectl.Kustomize == true {
		args = append(args, "--kustomize", manifest)
	} else {
		args = append(args, "--filename", manifest)
	}

	// Add extra args
	args = append(args, k.config.Kubectl.CreateArgs...)

	// Execute command
	output, err := k.cmd.RunCommand(k.path, args)
	if err != nil {
		_, ok := err.(*exec.ExitError)
		if ok {
			return nil, errors.New(string(output))
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
		err := yaml.Unmarshal([]byte(part), &objMap)
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
		err = yaml.Unmarshal([]byte(part), &obj)
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

