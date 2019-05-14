package helm

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/helm/pkg/helm"
)

func TestInstallChart(t *testing.T) {
	config := createFakeConfig()

	// Create the fake client.
	kubeClient := fake.NewSimpleClientset()
	helmClient := &helm.FakeClient{}

	client, err := create(config, configutil.TestNamespace, helmClient, kubeClient)
	if err != nil {
		t.Fatal(err)
	}

	helmConfig := &latest.HelmConfig{
		Chart: &latest.ChartConfig{
			Name: ptr.String("stable/nginx-ingress"),
		},
	}

	err = client.UpdateRepos()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.InstallChart("my-release", "", &map[interface{}]interface{}{}, helmConfig)
	if err != nil {
		t.Fatal(err)
	}

	// Upgrade
	_, err = client.InstallChart("my-release", "", &map[interface{}]interface{}{}, helmConfig)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAnalyzeError(t *testing.T) {
	// TODO
}
