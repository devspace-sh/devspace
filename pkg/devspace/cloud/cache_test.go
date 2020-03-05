package cloud

import (
	"regexp"
	"testing"
	"time"

	fakeclient "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client/testing"
	testconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	fakekubeconfig "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"k8s.io/client-go/tools/clientcmd/api"
)

type cacheSpaceTestCase struct {
	name string

	space          *latest.Space
	serviceAccount *latest.ServiceAccount
	spaces         map[int]*latest.SpaceCache

	expectedErr    string
	expectedSpaces map[int]*latest.SpaceCache
}

func TestCacheSpace(t *testing.T) {
	testCases := []cacheSpaceTestCase{
		cacheSpaceTestCase{
			name: "Save into nil cache",
			space: &latest.Space{
				Name:    "someSpace",
				SpaceID: 2,
			},
			serviceAccount: &latest.ServiceAccount{
				Namespace: "testNS",
			},
			expectedSpaces: map[int]*latest.SpaceCache{
				2: &latest.SpaceCache{
					Space: &latest.Space{
						Name:    "someSpace",
						SpaceID: 2,
					},
					ServiceAccount: &latest.ServiceAccount{
						Namespace: "testNS",
					},
					KubeContext: "devspace--somespace",
				},
			},
		},
		cacheSpaceTestCase{
			name: "Save existing space",
			space: &latest.Space{
				Name:    "someSpace2",
				SpaceID: 3,
			},
			spaces: map[int]*latest.SpaceCache{
				3: &latest.SpaceCache{
					LastResume: int64(1234),
				},
				1: &latest.SpaceCache{
					Space: &latest.Space{
						Name: "spaceFromBefore",
					},
					KubeContext: "contextInRawConfig",
				},
			},
			expectedSpaces: map[int]*latest.SpaceCache{
				3: &latest.SpaceCache{
					Space: &latest.Space{
						Name:    "someSpace2",
						SpaceID: 3,
					},
					LastResume:  int64(1234),
					KubeContext: "devspace--somespace2",
				},
				1: &latest.SpaceCache{
					Space: &latest.Space{
						Name: "spaceFromBefore",
					},
					KubeContext: "contextInRawConfig",
				},
			},
		},
	}

	for _, testCase := range testCases {
		provider := &provider{
			Provider: latest.Provider{
				Spaces: testCase.spaces,
			},
			kubeLoader: &fakekubeconfig.Loader{
				RawConfig: &api.Config{
					Contexts: map[string]*api.Context{
						"contextInRawConfig": &api.Context{},
					},
				},
			},
			loader: testconfig.NewLoader(&latest.Config{}),
		}
		err := provider.CacheSpace(testCase.space, testCase.serviceAccount)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		spacesAsYaml, err := yaml.Marshal(provider.Provider.Spaces)
		assert.NilError(t, err, "Error parsing spaces to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedSpaces)
		assert.NilError(t, err, "Error parsing spaces expection to yaml in testCase %s", testCase.name)
		lineWithTimestamp := regexp.MustCompile("(?m)[\r\n]+^.*expires.*$")
		spacesString := lineWithTimestamp.ReplaceAllString(string(spacesAsYaml), "")
		expectedString := lineWithTimestamp.ReplaceAllString(string(expectedAsYaml), "")
		assert.Equal(t, spacesString, expectedString, "Unexpected spaces in testCase %s", testCase.name)
	}
}

type getAndUpdateSpaceCacheTestCase struct {
	name string

	spaceID      int
	forceUpdate  bool
	spaces       map[int]*latest.SpaceCache
	clientSpaces []*latest.Space

	expectedErr        string
	expectedWasUpdated bool
	expectedSpace      *latest.SpaceCache
}

func TestGetAndUpdateCacheSpace(t *testing.T) {
	in1Hour := time.Now().Add(time.Hour).Unix()
	testCases := []getAndUpdateSpaceCacheTestCase{
		getAndUpdateSpaceCacheTestCase{
			name:    "Get saved space without update",
			spaceID: 6,
			spaces: map[int]*latest.SpaceCache{
				6: &latest.SpaceCache{
					Space: &latest.Space{
						SpaceID: 6,
						Name:    "expectedSpace",
					},
					Expires: in1Hour,
				},
			},
			expectedSpace: &latest.SpaceCache{
				Space: &latest.Space{
					SpaceID: 6,
					Name:    "expectedSpace",
				},
				Expires: in1Hour,
			},
		},
		getAndUpdateSpaceCacheTestCase{
			name:    "Get from client",
			spaceID: 4,
			clientSpaces: []*latest.Space{
				&latest.Space{
					SpaceID: 4,
					Name:    "clientSpace",
					Cluster: &latest.Cluster{},
				},
			},
			expectedSpace: &latest.SpaceCache{
				Space: &latest.Space{
					SpaceID: 4,
					Name:    "clientSpace",
					Cluster: &latest.Cluster{},
				},
				ServiceAccount: &latest.ServiceAccount{},
				KubeContext:    "devspace--clientspace",
			},
			expectedWasUpdated: true,
		},
	}

	for _, testCase := range testCases {
		provider := &provider{
			Provider: latest.Provider{
				Spaces: testCase.spaces,
			},
			kubeLoader: &fakekubeconfig.Loader{
				RawConfig: &api.Config{
					Contexts: map[string]*api.Context{
						"contextInRawConfig": &api.Context{},
					},
				},
			},
			loader: testconfig.NewLoader(&latest.Config{}),
			client: &fakeclient.CloudClient{
				Spaces: testCase.clientSpaces,
			},
		}
		space, wasUpdated, err := provider.GetAndUpdateSpaceCache(testCase.spaceID, testCase.forceUpdate)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
		assert.Equal(t, wasUpdated, testCase.expectedWasUpdated, "Unexpected updated bool in testCase %s", testCase.name)

		spaceAsYaml, err := yaml.Marshal(space)
		assert.NilError(t, err, "Error parsing space to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedSpace)
		assert.NilError(t, err, "Error parsing space expection to yaml in testCase %s", testCase.name)
		lineWithTimestamp := regexp.MustCompile("(?m)[\r\n]+^.*expires.*$")
		spaceString := lineWithTimestamp.ReplaceAllString(string(spaceAsYaml), "")
		expectedString := lineWithTimestamp.ReplaceAllString(string(expectedAsYaml), "")
		assert.Equal(t, spaceString, expectedString, "Unexpected space in testCase %s", testCase.name)
	}
}
