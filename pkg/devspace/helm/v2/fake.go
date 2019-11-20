package v2

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
	k8shelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	hapi_release5 "k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

// FakeClient implements Interface
type FakeClient struct {
	helm helm.Interface
	kube kubernetes.Interface
}

// NewFakeClient creates a new fake client
func NewFakeClient(kubeClient kubernetes.Interface, tillerNamespace string) *FakeClient {
	helmClient := &helm.FakeClient{}

	return &FakeClient{
		helm: helmClient,
		kube: kubeClient,
	}
}

// UpdateRepos implements interface
func (f *FakeClient) UpdateRepos() error {
	return nil
}

// DeleteRelease deletes a helm release and optionally purges it
func (f *FakeClient) DeleteRelease(releaseName string, purge bool) (*rls.UninstallReleaseResponse, error) {
	return f.helm.DeleteRelease(releaseName, k8shelm.DeletePurge(purge))
}

// ListReleases lists all helm releases
func (f *FakeClient) ListReleases() (*rls.ListReleasesResponse, error) {
	return f.helm.ListReleases()
}

// InstallChart implements interface
func (f *FakeClient) InstallChart(releaseName string, releaseNamespace string, values *map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*hapi_release5.Release, error) {
	chart := &chart.Chart{
		Metadata: &chart.Metadata{Name: "test-chart"},
	}

	releaseExists := ReleaseExists(f.helm, releaseName)
	if releaseExists {
		upgradeResponse, err := f.helm.UpdateRelease(
			releaseName,
			"random_path",
		)

		if err != nil {
			return nil, err
		}

		return upgradeResponse.GetRelease(), nil
	}

	installResponse, err := f.helm.InstallReleaseFromChart(
		chart,
		releaseNamespace,
		k8shelm.ReleaseName(releaseName),
		k8shelm.InstallReuseName(true),
	)
	if err != nil {
		return nil, err
	}

	return installResponse.GetRelease(), nil
}
