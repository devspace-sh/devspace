package cloud

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	fakeclient "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client/testing"
	testconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	fakekubeconfig "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gotest.tools/assert"
	"k8s.io/client-go/tools/clientcmd/api"
)

type printSpacesTestCase struct {
	name string

	cluster      string
	nameParam    string
	all          bool
	clientSpaces []*latest.Space

	expectedErr string
}

func TestPrintSpaces(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
		Hasura: token.Hasura{
			AccountID: "1",
		},
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}
	validToken := "." + validEncodedClaim + "."

	testCases := []printSpacesTestCase{
		printSpacesTestCase{
			name: "No spaces",
		},
		printSpacesTestCase{
			name:      "Print spaces",
			cluster:   "printThisCluster",
			nameParam: "printThisSpace",
			clientSpaces: []*latest.Space{
				&latest.Space{
					Name: "skip",
				},
				&latest.Space{
					Name: "printThisSpace",
					Cluster: &latest.Cluster{
						Name: "Skip",
					},
				},
				&latest.Space{
					Name: "printThisSpace",
					Cluster: &latest.Cluster{
						Name: "printThisCluster",
					},
					Owner: &latest.Owner{
						Name:    "WrongID",
						OwnerID: 2,
					},
				},
				&latest.Space{
					Name: "printThisSpace",
					Cluster: &latest.Cluster{
						Name: "printThisCluster",
					},
					Owner: &latest.Owner{
						Name:    "CorrectID",
						OwnerID: 1,
					},
					Created: "2006-01-02T15:04:05",
				},
			},
		},
	}

	for _, testCase := range testCases {
		provider := &provider{
			Provider: latest.Provider{
				Name: "providerName",
			},
			kubeLoader: &fakekubeconfig.Loader{
				RawConfig: &api.Config{},
			},
			loader: testconfig.NewLoader(&latest.Config{}),
			client: &fakeclient.CloudClient{
				Spaces: testCase.clientSpaces,
				Token:  validToken,
			},
			log: log.Discard,
		}

		err := provider.PrintSpaces(testCase.cluster, testCase.nameParam, testCase.all)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type printTokenTestCase struct {
	name string

	spaces  map[int]*latest.SpaceCache
	spaceID int

	expectedErr string
}

func TestPrintToken(t *testing.T) {
	in1Hour := time.Now().Add(time.Hour).Unix()
	testCases := []printTokenTestCase{
		printTokenTestCase{
			name:    "Resume and print space in cache",
			spaceID: 1,
			spaces: map[int]*latest.SpaceCache{
				1: &latest.SpaceCache{
					Space: &latest.Space{
						Cluster: &latest.Cluster{},
					},
					LastResume:     time.Now().Add(time.Hour * (-1)).Unix(),
					Expires:        in1Hour,
					ServiceAccount: &latest.ServiceAccount{},
				},
			},
		},
	}

	for _, testCase := range testCases {
		provider := &provider{
			Provider: latest.Provider{
				Name:   "providerName",
				Spaces: testCase.spaces,
			},
			kubeLoader: &fakekubeconfig.Loader{
				RawConfig: &api.Config{},
			},
			client: &fakeclient.CloudClient{},
			loader: testconfig.NewLoader(&latest.Config{}),
			log:    log.Discard,
		}

		err := provider.PrintToken(testCase.spaceID)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}
