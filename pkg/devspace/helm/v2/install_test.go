package v2

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
	helmchartutil "k8s.io/helm/pkg/chartutil"
	helmenvironment "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/repo"
)

type installChartTestCase struct {
	name string

	files            map[string]interface{}
	releaseName      string
	releaseNamespace string
	values           map[interface{}]interface{}
	helmConfig       *latest.HelmConfig
	releases         *[]*release.Release

	expectedErr     bool
	expectedRelease *types.Release
}

func TestInstallChart(t *testing.T) {
	testCases := []installChartTestCase{
		{
			name: "Relative path to chart not there",
			helmConfig: &latest.HelmConfig{
				Chart: &latest.ChartConfig{
					Name: "./notThere",
				},
			},
			expectedErr: true,
		},
		{
			name: "Install dir chart",
			helmConfig: &latest.HelmConfig{
				Chart: &latest.ChartConfig{
					Name: "myChart",
				},
			},
			files: map[string]interface{}{
				"myChart/Chart.yaml": chart.Metadata{
					Name: "myChart",
				},
			},
			expectedRelease: &types.Release{
				Name:         "myChart",
				Namespace:    "testNamespace",
				Status:       "12345",
				Version:      0,
				LastDeployed: time.Unix(12345, 0),
			},
		},
		{
			name:        "Install dir chart in repository dir",
			releaseName: "myRelease",
			helmConfig: &latest.HelmConfig{
				Chart: &latest.ChartConfig{
					Name: "myChart",
				},
				Timeout: ptr.Int64(time.Now().Add(time.Hour).Unix()),
			},
			files: map[string]interface{}{
				"repository/myChart/Chart.yaml": chart.Metadata{
					Name: "myChart",
				},
				"repository/myChart/requirements.yaml": &helmchartutil.Requirements{
					Dependencies: []*helmchartutil.Dependency{
						&helmchartutil.Dependency{
							Name: "dep1",
						},
					},
				},
				"repository/repositories.yaml": repo.RepoFile{
					APIVersion: "a",
				},
				"repository/myChart/charts/missingDep/Chart.yaml": chart.Metadata{
					Name: "dep1",
				},
			},
			releases: &[]*release.Release{
				&release.Release{
					Name: "myRelease",
				},
			},
			expectedRelease: &types.Release{
				Name:         "myRelease",
				Namespace:    "myRelease",
				Status:       "12345",
				Version:      0,
				LastDeployed: time.Unix(12345, 0),
			},
		},
	}

	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	for _, testCase := range testCases {
		for path, content := range testCase.files {
			asJSON, err := json.Marshal(content)
			assert.NilError(t, err, "Error parsing content to json in testCase %s", testCase.name)
			if content == "" {
				asJSON = []byte{}
			}
			err = fsutil.WriteToFile(asJSON, path)
			assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
		}

		client := &client{
			Settings: &helmenvironment.EnvSettings{
				Home: helmpath.Home(dir),
			},
			kubectl: &fakekube.Client{},
			helm: &fakeHelm{
				releases: testCase.releases,
			},
		}

		release, err := client.InstallChart(testCase.releaseName, testCase.releaseNamespace, testCase.values, testCase.helmConfig)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		releaseAsYaml, err := yaml.Marshal(release)
		assert.NilError(t, err, "Error parsing release to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedRelease)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(releaseAsYaml), string(expectedAsYaml), "Unexpected release in testCase %s", testCase.name)

		err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
			os.RemoveAll(path)
			return nil
		})
		assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
	}
}
