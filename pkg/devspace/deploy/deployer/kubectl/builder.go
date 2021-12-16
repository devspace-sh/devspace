package kubectl

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/util/constraint"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/command"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Builder is the manifest builder interface
type Builder interface {
	Build(manifest string, executor RunCommand) ([]*unstructured.Unstructured, error)
}

type RunCommand func(path string, args []string) ([]byte, error)

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

func (k *kustomizeBuilder) Build(manifest string, cmd RunCommand) ([]*unstructured.Unstructured, error) {
	args := []string{"build", manifest}
	args = append(args, k.config.Kubectl.KustomizeArgs...)

	// Execute command
	k.log.Infof("Render manifests with 'kustomize %s'", strings.Join(args, " "))
	output, err := cmd(k.path, args)
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
	path        string
	config      *latest.DeploymentConfig
	context     string
	namespace   string
	isInCluster bool
}

// NewKubectlBuilder creates a new kubectl manifest builder
func NewKubectlBuilder(path string, config *latest.DeploymentConfig, context, namespace string, isInCluster bool) Builder {
	return &kubectlBuilder{
		path:        path,
		config:      config,
		context:     context,
		namespace:   namespace,
		isInCluster: isInCluster,
	}
}

// this function is called in Build function
// to decide the --dry-run value
var useOldDryRun = func(path string) (bool, error) {
	// compare kubectl version for --dry-run flag value
	out, err := command.NewStreamCommand(path, []string{"version", "--client", "--short"}).Output()
	if err != nil {
		return false, err
	}

	v1, err := constraint.NewVersion(strings.TrimPrefix(strings.TrimSpace(string(out)), "Client Version: v"))
	if err != nil {

		return false, err
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

func (k *kubectlBuilder) Build(manifest string, cmd RunCommand) ([]*unstructured.Unstructured, error) {
	args := []string{"create"}
	if k.context != "" && !k.isInCluster {
		args = append(args, "--context", k.context)
	}
	if k.namespace != "" {
		args = append(args, "--namespace", k.namespace)
	}

	// decides which --dry-run value is to be used
	uodr, err := useOldDryRun(k.path)
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
	output, err := cmd(k.path, args)
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
