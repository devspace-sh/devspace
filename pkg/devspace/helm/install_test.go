package helm

import (
	"fmt"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	helmchartutil "k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"gotest.tools/assert"
)

type checkDependenciesTestCase struct {
	name string

	dependenciesInChart        []*chart.Chart
	dependenciesInRequirements []*helmchartutil.Dependency

	expectedErr string
}

func TestCheckDependencies(t *testing.T) {
	startTime := time.Now()
	testCases := []checkDependenciesTestCase{
		checkDependenciesTestCase{
			name:                       "Matching dependencies in chart and requirements",
			dependenciesInChart:        []*chart.Chart{&chart.Chart{Metadata: &chart.Metadata{Name: "MatchingDep"}}},
			dependenciesInRequirements: []*helmchartutil.Dependency{&helmchartutil.Dependency{Name: "MatchingDep"}},
		},
		checkDependenciesTestCase{
			name:                       "Requirements has dependency and that the chart has not",
			dependenciesInChart:        []*chart.Chart{&chart.Chart{Metadata: &chart.Metadata{Name: "ChartDep"}}},
			dependenciesInRequirements: []*helmchartutil.Dependency{&helmchartutil.Dependency{Name: "ReqDep"}},
			expectedErr:                "found in requirements.yaml, but missing in charts/ directory: ReqDep",
		},
	}

	for _, testCase := range testCases {
		ch := &chart.Chart{
			Dependencies: testCase.dependenciesInChart,
		}
		reqs := &helmchartutil.Requirements{
			Dependencies: testCase.dependenciesInRequirements,
		}

		err := checkDependencies(ch, reqs)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error checking dependencies in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error checking dependencies in testCase %s", testCase.name)
		}
	}
	t.Log("TestCheckDependencies needed " + time.Since(startTime).String())
	t.Fatal("This fatal is to show the logs")
}

func TestInstallChart(t *testing.T) {
	startTime := time.Now()
	config := createFakeConfig()

	// Create the fake client.
	kubeClient := fake.NewSimpleClientset()
	helmClient := &helm.FakeClient{}

	client, err := create(config, configutil.TestNamespace, helmClient, kubeClient, log.GetInstance())
	if err != nil {
		t.Fatal(err)
	}

	helmConfig := &latest.HelmConfig{
		Chart: &latest.ChartConfig{
			Name: ptr.String("stable/nginx-ingress"),
		},
	}

	err = client.UpdateRepos(log.GetInstance())
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
	t.Log("TestInstallChart needed " + time.Since(startTime).String())
	t.Fatal("This fatal is to show the logs")
}

type analyzeErrorTestCase struct {
	name string

	inputErr    error
	namespace   string
	createdPods []*k8sv1.Pod

	expectedErr string
}

func TestAnalyzeError(t *testing.T) {
	startTime := time.Now()
	testCases := []analyzeErrorTestCase{
		analyzeErrorTestCase{
			name:        "Test analyze no-timeout error",
			inputErr:    fmt.Errorf("Some error"),
			expectedErr: "Some error",
		},
		analyzeErrorTestCase{
			name:      "Test analyze timeout error",
			inputErr:  fmt.Errorf("timed out waiting"),
			namespace: "testNS",
		},
	}

	for _, testCase := range testCases {
		config := createFakeConfig()

		// Create the fake client.
		kubeClient := fake.NewSimpleClientset()
		helmClient := &helm.FakeClient{}

		for _, pod := range testCase.createdPods {
			_, err := kubeClient.CoreV1().Pods(testCase.namespace).Create(pod)
			assert.NilError(t, err, "Error creating testPod in testCase %s", testCase.name)
		}

		client, err := create(config, configutil.TestNamespace, helmClient, kubeClient, &log.DiscardLogger{})
		if err != nil {
			t.Fatal(err)
		}

		err = client.analyzeError(testCase.inputErr, testCase.namespace)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error analyzing error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error returned in testCase %s", testCase.name)
		}
	}
	t.Log("TestAnalyzeError needed " + time.Since(startTime).String())
	t.Fatal("This fatal is to show the logs")
}
