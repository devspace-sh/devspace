package v2

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"k8s.io/client-go/kubernetes/fake"
)

//@MoreTest
//Coverage 51% is not enough

func TestFakeInstallChart(t *testing.T) {
	// Create the fake client.
	kubeClient := fake.NewSimpleClientset()
	fakeClient := NewFakeClient(kubeClient, configutil.TestNamespace)

	err := fakeClient.UpdateRepos()
	if err != nil {
		t.Fatal(err)
	}

	// Install release
	release, err := fakeClient.InstallChart("test-release", configutil.TestNamespace, &map[interface{}]interface{}{}, &latest.HelmConfig{
		Chart: &latest.ChartConfig{
			Name: "stable/nginx-ingress",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if release.Name != "test-release" {
		t.Fatalf("Expected release name test-release, got %s", release.Name)
	}

	// Update release
	release, err = fakeClient.InstallChart("test-release", configutil.TestNamespace, &map[interface{}]interface{}{}, &latest.HelmConfig{
		Chart: &latest.ChartConfig{
			Name: "stable/nginx-ingress",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if release.Name != "test-release" {
		t.Fatalf("Expected release name test-release, got %s", release.Name)
	}

	releases, err := fakeClient.ListReleases()
	if err != nil {
		t.Fatal(err)
	}
	if len(releases.Releases) == 0 || releases.Releases[0].Name != "test-release" {
		t.Fatalf("Wrong amount of releases returned: %#+v", releases.Releases)
	}

	// Delete release
	_, err = fakeClient.DeleteRelease("test-release", true)
	if err != nil {
		t.Fatal(err)
	}
}
