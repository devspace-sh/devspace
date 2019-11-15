package v3

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"helm.sh/helm/v3/pkg/action"
)

type v3Client struct {
	kubeClient kubectl.Client
}

// NewClient creates a new helm v3 client
func NewClient(kubeClient kubectl.Client) (helm.Client, error) {
	return &v3Client{
		kubeClient: kubeClient,
	}, nil
}

func (client *v3Client) InstallChart(releaseName string, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*helm.Release, error) {
	return nil, nil
}

func (client *v3Client) DeleteRelease(releaseName string, purge bool) error {
	action.NewUninstall(nil)

	return nil
}

func (client *v3Client) ListReleases() ([]*helm.Release, error) {
	return nil, nil
}
