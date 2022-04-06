package compose

import (
	"fmt"
	"path/filepath"

	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func (cb *configBuilder) AddSecret(dockerCompose *composetypes.Project, service composetypes.ServiceConfig) error {
	var pipelines map[string]*latest.Pipeline
	for secretName, secret := range dockerCompose.Secrets {
		if pipelines == nil {
			pipelines = map[string]*latest.Pipeline{}
		}

		devSecretStep, err := createSecretPipeline(secretName, cb.workingDir, secret)
		if err != nil {
			return err
		}

		pipelines["dev"] = devSecretStep
		pipelines["purge"] = deleteSecretPipeline(secretName)
	}

	cb.config.Pipelines = pipelines
	return nil
}

func createSecretPipeline(name string, cwd string, secret composetypes.SecretConfig) (*latest.Pipeline, error) {
	file, err := filepath.Rel(cwd, filepath.Join(cwd, secret.File))
	if err != nil {
		return nil, err
	}

	return &latest.Pipeline{
		Run: fmt.Sprintf(`kubectl create secret generic %s --namespace=${devspace.namespace} --dry-run=client --from-file=%s=%s -o yaml | kubectl apply -f -
run_default_pipeline dev`, name, name, filepath.ToSlash(file)),
	}, nil
}

func deleteSecretPipeline(name string) *latest.Pipeline {
	return &latest.Pipeline{
		Run: fmt.Sprintf(`run_default_pipeline purge
kubectl delete secret %s --namespace=${devspace.namespace} --ignore-not-found`, name),
	}
}
