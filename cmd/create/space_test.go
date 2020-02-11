package create

/*import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cloudpkg "github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"gotest.tools/assert"
)

type customGraphqlClient struct {
	responses []interface{}
}

func (q *customGraphqlClient) GrapqhlRequest(p *cloudpkg.Provider, request string, vars map[string]interface{}, response interface{}) error {
	if len(q.responses) == 0 {
		return fmt.Errorf("Custom graphQL server error")
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

type createSpaceTestCase struct {
	name string

	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider
	args             []string
	answers          []string

	activeFlag   bool
	providerFlag string
	clusterFlag  string

	expectedErr string
}

func TestRunCreateSpace(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}

	testCases := []createSpaceTestCase{
		createSpaceTestCase{
			name:         "Provider doesn't Exist",
			providerFlag: "Doesn'tExist",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
				},
			},
			expectedErr: "Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: SomeProvider ",
		},
		createSpaceTestCase{
			name:         "Projects can't be retrieved",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
					Key:  "someKey",
				},
			},
			expectedErr:    "get projects: Custom graphQL server error",
		},
		createSpaceTestCase{
			name:         "New project can't be created",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
					Key:  "someKey",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{
					Projects: []*cloudlatest.Project{},
				},
			},
			expectedErr:    "Custom graphQL server error",
		},
		createSpaceTestCase{
			name:         "Unparsable cluster name",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
					Key:  "someKey",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{Projects: []*cloudlatest.Project{&cloudlatest.Project{}}},
			},
			clusterFlag:    "a:b:c",
			expectedErr:    "Error parsing cluster name a:b:c: Expected : only once",
		},
		createSpaceTestCase{
			name:         "Can't get clusters",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
					Key:  "someKey",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{Projects: []*cloudlatest.Project{&cloudlatest.Project{}}},
			},
			expectedErr:    "get clusters: Custom graphQL server error",
		},
		createSpaceTestCase{
			name:         "No clusters",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name: "SomeProvider",
					Key:  "someKey",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{Projects: []*cloudlatest.Project{&cloudlatest.Project{}}},
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{Clusters: []*cloudlatest.Cluster{}},
			},
			expectedErr:    "Cannot create space, because no cluster was found",
		},
		createSpaceTestCase{
			name:         "Question 1 is not possible",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "SomeProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{Projects: []*cloudlatest.Project{&cloudlatest.Project{}}},
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{Clusters: []*cloudlatest.Cluster{
					&cloudlatest.Cluster{Owner: &cloudlatest.Owner{}},
					&cloudlatest.Cluster{},
				}},
			},
			expectedErr:    "Cannot ask question 'Which cluster should the space created in?' because logger level is too low",
		},
		createSpaceTestCase{
			name:         "Question 2 is not possible",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "SomeProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{Projects: []*cloudlatest.Project{&cloudlatest.Project{}}},
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{Clusters: []*cloudlatest.Cluster{
					&cloudlatest.Cluster{},
					&cloudlatest.Cluster{},
				}},
			},
			expectedErr:    "Cannot ask question 'Which hosted DevSpace cluster should the space created in?' because logger level is too low",
		},
		createSpaceTestCase{
			name:         "Select non existent cluster",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "SomeProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{Projects: []*cloudlatest.Project{&cloudlatest.Project{}}},
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{Clusters: []*cloudlatest.Cluster{
					&cloudlatest.Cluster{},
					&cloudlatest.Cluster{},
				}},
			},
			answers: []string{"notthere"},
			expectedErr:    "No cluster selected",
		},
		createSpaceTestCase{
			name:         "Space creation fails with server error",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "SomeProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{Projects: []*cloudlatest.Project{&cloudlatest.Project{}}},
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{Clusters: []*cloudlatest.Cluster{
					&cloudlatest.Cluster{Owner: &cloudlatest.Owner{}},
				}},
			},
			expectedErr:    "create space: Custom graphQL server error",
		},
		createSpaceTestCase{
			name:         "Get space fails after creation",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "SomeProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{Projects: []*cloudlatest.Project{&cloudlatest.Project{}}},
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{Clusters: []*cloudlatest.Cluster{
					&cloudlatest.Cluster{Owner: &cloudlatest.Owner{}, Name: "cluster1"},
					&cloudlatest.Cluster{Owner: &cloudlatest.Owner{}, Name: "cluster2"},
				}},
				struct {
					CreateSpace interface{} `json:"manager_createSpace"`
				}{CreateSpace: struct{ SpaceID int }{SpaceID: 1}},
			},
			answers:        []string{"cluster1"},
			expectedErr:    "get space: Custom graphQL server error",
		},
		createSpaceTestCase{
			name:         "Get serviceaccount fails after creation",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "SomeProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{[]*cloudlatest.Project{&cloudlatest.Project{}}},
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{[]*cloudlatest.Cluster{
					&cloudlatest.Cluster{Owner: &cloudlatest.Owner{}, Name: "cluster1"},
					&cloudlatest.Cluster{Name: "cluster2"},
				}},
				struct {
					CreateSpace interface{} `json:"manager_createSpace"`
				}{CreateSpace: struct{ SpaceID int }{SpaceID: 1}},
				struct {
					Space interface{} `json:"space_by_pk"`
				}{
					struct {
						KubeContext interface{} `json:"kube_context"`
						Owner       interface{} `json:"account"`
					}{struct{Cluster interface{} `json:"cluster"`}{struct{}{}}, struct{}{}},
				},
			},
			answers:        []string{DevSpaceCloudHostedCluster},
			expectedErr:    "get serviceaccount: Custom graphQL server error",
		},
		createSpaceTestCase{
			name:         "Undecodable CACert",
			providerFlag: "SomeProvider",
			providerList: []*cloudlatest.Provider{
				&cloudlatest.Provider{
					Name:  "SomeProvider",
					Key:   "someKey",
					Token: "." + validEncodedClaim + ".",
				},
			},
			graphQLResponses: []interface{}{
				struct {
					Projects []*cloudlatest.Project `json:"project"`
				}{[]*cloudlatest.Project{&cloudlatest.Project{}}},
				struct {
					Clusters []*cloudlatest.Cluster `json:"cluster"`
				}{[]*cloudlatest.Cluster{
					&cloudlatest.Cluster{Name: "cluster2"},
					&cloudlatest.Cluster{Name: "cluster3"},
				}},
				struct {
					CreateSpace interface{} `json:"manager_createSpace"`
				}{CreateSpace: struct{ SpaceID int }{SpaceID: 1}},
				struct {
					Space interface{} `json:"space_by_pk"`
				}{
					struct {
						KubeContext interface{} `json:"kube_context"`
						Owner       interface{} `json:"account"`
					}{struct{Cluster interface{} `json:"cluster"`}{struct{}{}}, struct{}{}},
				},
				struct {ServiceAccount *cloudlatest.ServiceAccount `json:"manager_serviceAccount"`}{&cloudlatest.ServiceAccount{
					CaCert: "undecodable",
				}},
			},
			answers:        []string{"cluster2"},
			expectedErr:    "update kube context: illegal base64 data at input byte 8",
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunCreateSpace(t, testCase)
	}
}

func testRunCreateSpace(t *testing.T, testCase createSpaceTestCase) {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

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

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	providerConfig, err := cloudconfig.Load()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	if len(testCase.args) == 0 {
		testCase.args = []string{""}
	}
	err = (&spaceCmd{
		Active:   testCase.activeFlag,
		Provider: testCase.providerFlag,
		Cluster:  testCase.clusterFlag,
	}).RunCreateSpace(nil, testCase.args)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}
*/