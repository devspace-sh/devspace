package v2

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"k8s.io/helm/pkg/helm"
	k8shelm "k8s.io/helm/pkg/helm"
	helmenvironment "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/proto/hapi/services"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/repo"
)

type fakeHelm struct {
	helm.Interface
	releases *[]*release.Release
}

func (f *fakeHelm) ListReleases(opts ...k8shelm.ReleaseListOption) (*rls.ListReleasesResponse, error) {
	if f.releases == nil {
		return nil, nil
	}
	return &rls.ListReleasesResponse{
		Releases: *f.releases,
	}, nil
}

func (f *fakeHelm) InstallReleaseFromChart(chart *chart.Chart, namespace string, opts ...k8shelm.InstallOption) (*services.InstallReleaseResponse, error) {
	return &services.InstallReleaseResponse{
		Release: &release.Release{
			Name:      chart.Metadata.Name,
			Namespace: namespace,
			Chart:     chart,
			Info: &release.Info{
				Status: &release.Status{
					Code: 12345,
				},
				LastDeployed: &timestamp.Timestamp{
					Seconds: 12345,
				},
			},
		},
	}, nil
}

func (f *fakeHelm) UpdateRelease(rlsName string, chStr string, opts ...k8shelm.UpdateOption) (*services.UpdateReleaseResponse, error) {
	if strings.Index(rlsName, "timeout") != -1 {
		return nil, errors.Errorf("timed out waiting")
	} else if strings.Index(rlsName, "error") != -1 {
		return nil, errors.Errorf("error")
	} else {
		return &services.UpdateReleaseResponse{
			Release: &release.Release{
				Name:      rlsName,
				Namespace: rlsName,
				Chart: &chart.Chart{
					Metadata: &chart.Metadata{
						Name: chStr,
					},
				},
				Info: &release.Info{
					Status: &release.Status{
						Code: 12345,
					},
					LastDeployed: &timestamp.Timestamp{
						Seconds: 12345,
					},
				},
			},
		}, nil
	}
}

type updateReposTestCase struct {
	name string

	files map[string]interface{}

	expectedErr bool
}

func TestUpdateRepos(t *testing.T) {
	testCases := []updateReposTestCase{
		{
			name: "No repos",
			files: map[string]interface{}{
				"repository/repositories.yaml": repo.RepoFile{
					APIVersion: "a",
				},
			},
		},
		{
			name: "Fail to update 1 repos",
			files: map[string]interface{}{
				"repository/repositories.yaml": repo.RepoFile{
					APIVersion: "a",
					Repositories: []*repo.Entry{
						&repo.Entry{
							URL: "http://definitlyNotThere",
						},
					},
				},
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
			log: &log.FakeLogger{},
		}

		err := client.UpdateRepos()

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
			os.RemoveAll(path)
			return nil
		})
		assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
	}
}

type releaseExistsTestCase struct {
	name string

	releases []*release.Release
	needle   string

	expectedExists bool
}

func TestReleaseExists(t *testing.T) {
	testCases := []releaseExistsTestCase{
		{
			name: "Exists",
			releases: []*release.Release{
				&release.Release{
					Name: "NotNeedle",
				},
				&release.Release{
					Name: "Needle",
				},
			},
			needle:         "Needle",
			expectedExists: true,
		},
		{
			name: "Not exists",
			releases: []*release.Release{
				&release.Release{
					Name: "NotNeedle",
				},
			},
			needle: "Needle",
		},
	}

	for _, testCase := range testCases {
		helmClient := &fakeHelm{
			&k8shelm.Client{},
			&testCase.releases,
		}
		exists := ReleaseExists(helmClient, testCase.needle)

		assert.Equal(t, exists, testCase.expectedExists, "Unexpected result in testCase %s", testCase.name)
	}
}

type listReleasesTestCase struct {
	name string

	releases *[]*release.Release

	expectedErr      bool
	expectedReleases []*types.Release
}

func TestListRelease(t *testing.T) {
	testCases := []listReleasesTestCase{
		{
			name: "Releases are nil",
		},
		{
			name: "List one releases",
			releases: &[]*release.Release{
				&release.Release{
					Name:      "ListThis",
					Namespace: "ListThisNamespace",
					Version:   321,
					Info: &release.Info{
						Status: &release.Status{
							Code: release.Status_Code(123),
						},
						LastDeployed: &timestamp.Timestamp{
							Seconds: 1234,
						},
					},
				},
			},
			expectedReleases: []*types.Release{
				{
					Name:         "ListThis",
					Namespace:    "ListThisNamespace",
					Status:       "123",
					Version:      321,
					LastDeployed: time.Unix(1234, 0),
				},
			},
		},
	}

	for _, testCase := range testCases {
		client := &client{
			helm: &fakeHelm{
				&k8shelm.Client{},
				testCase.releases,
			},
		}
		releases, err := client.ListReleases()

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		releasesAsYaml, err := yaml.Marshal(releases)
		assert.NilError(t, err, "Error parsing releases to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedReleases)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(releasesAsYaml), string(expectedAsYaml), "Unexpected releases in testCase %s", testCase.name)
	}
}
