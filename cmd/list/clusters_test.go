package list

/*import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	homedir "github.com/mitchellh/go-homedir"

	"gotest.tools/assert"
)

type customGraphqlClient struct {
	responses []interface{}
}

func (q *customGraphqlClient) GrapqhlRequest(p *cloudpkg.Provider, request string, vars map[string]interface{}, response interface{}) error {
	if len(q.responses) == 0 {
		panic("Not enough responses. Need response for: " + request)
	}
	currentResponse := q.responses[0]
	q.responses = q.responses[1:]

	errorResponse, isError := currentResponse.(error)
	if isError {
		return errorResponse
	}
	buf, err := json.Marshal(currentResponse)
	if err != nil {
		panic(fmt.Sprintf("Cannot encode response. %d responses left", len(q.responses)))
	}
	json.NewDecoder(bytes.NewReader(buf)).Decode(&response)

	return nil
}

type listClustersTestCase struct {
	name string

	providerFlag     string
	allFlag          bool
	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider

	expectTablePrint bool
	expectedHeader   []string
	expectedValues   [][]string
	expectedErr      string
}

func TestListClusters(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

	testCases := []listClustersTestCase{
		listClustersTestCase{
			name:         "No clusters",
			providerFlag: "app.devspace.com",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "app.devspace.com",
					Key:  "someKey",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{
					Clusters: []*cloudlatest.Cluster{},
				},
			},
		},
		listClustersTestCase{
			name:         "One cluster",
			providerFlag: "app.devspace.com",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "app.devspace.com",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{
					Clusters: []*cloudlatest.Cluster{
						&cloudlatest.Cluster{
							ClusterID:    1,
							Server:       ptr.String("someServer"),
							Name:         "someName",
							EncryptToken: true,
							Owner: &cloudlatest.Owner{
								OwnerID: 1,
								Name:    "someOwner",
							},
						},
						&cloudlatest.Cluster{
							ClusterID:    2,
							Server:       ptr.String("someServer2"),
							Name:         "someName2",
							EncryptToken: true,
						},
					},
				},
			},
			expectedHeader: []string{"ID", "Name", "Owner", "Created"},
			expectedValues: [][]string{
				[]string{"1", "someOwner:someName", "someOwner", ""},
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

	homedir, err := homedir.Dir()
	assert.NilError(t, err, "Error getting homedir")
	relDir, err := filepath.Rel(homedir, dir)
	assert.NilError(t, err, "Error getting relative dir path")
	cloudconfig.DevSpaceProvidersConfigPath = filepath.Join(relDir, "Doesn'tExist")
	cloudconfig.LegacyDevSpaceCloudConfigPath = filepath.Join(relDir, "Doesn'tExist")

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	log.SetInstance(log.Discard)

	for _, testCase := range testCases {
		testListClusters(t, testCase)
	}
}

func testListClusters(t *testing.T, testCase listClustersTestCase) {
	log.SetFakePrintTable(func(s log.Logger, header []string, values [][]string) {
		assert.Assert(t, testCase.expectTablePrint || len(testCase.expectedHeader)+len(testCase.expectedValues) > 0, "PrintTable unexpectedly called in testCase %s", testCase.name)
		assert.Equal(t, reflect.DeepEqual(header, testCase.expectedHeader), true, "Unexpected header in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedHeader, header)
		assert.Equal(t, reflect.DeepEqual(values, testCase.expectedValues), true, "Unexpected values in testCase %s. Expected:%v\nActual:%v", testCase.name, testCase.expectedValues, values)
	})

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	providerConfig, err := cloudconfig.Load()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	err = (&clustersCmd{
		All:      testCase.allFlag,
		Provider: testCase.providerFlag,
	}).RunListClusters(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
