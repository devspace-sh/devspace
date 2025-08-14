package analyze

import (
	"context"
	"fmt"
	"testing"
	"time"

	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/mgutz/ansi"
	"gopkg.in/yaml.v3"
	"gotest.tools/assert"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type analyzeTestCase struct {
	name string

	namespace string
	noWait    bool

	expectedErr string
}

func TestAnalyze(t *testing.T) {
	testCases := []analyzeTestCase{
		{},
	}

	for _, testCase := range testCases {
		kubeClient := &fakekube.Client{
			Client: fake.NewSimpleClientset(),
		}
		analyzer := NewAnalyzer(kubeClient, log.Discard)

		err := analyzer.Analyze(testCase.namespace, Options{Wait: testCase.noWait})

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type createReportTestCase struct {
	name string

	kubeNamespaces   []string
	kubePods         map[string][]k8sv1.Pod
	kubeReplicasets  map[string][]appsv1.ReplicaSet
	kubeStatefulsets map[string][]appsv1.StatefulSet
	kubeEvents       map[string][]k8sv1.Event

	namespace string
	wait      bool

	expectedErr    string
	expectedReport []*ReportItem
}

/*
Part of this function is untestable right now because the helper function events uses the RestClient of the KubeClient.
The fake client returns nil. Therefore it's not possible to let events return problems.
*/
func TestCreateReport(t *testing.T) {
	testCases := []createReportTestCase{
		{
			name:           "Nothing to report",
			wait:           true,
			kubeNamespaces: []string{"ns1"},
			kubeReplicasets: map[string][]appsv1.ReplicaSet{
				"ns1": {
					{
						Spec: appsv1.ReplicaSetSpec{},
					},
				},
			},
		},
		{
			name:           "Error in pods",
			wait:           true,
			kubeNamespaces: []string{"ns1"},
			kubePods: map[string][]k8sv1.Pod{
				"ns1": {
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "ErrorPod",
						},
						Status: k8sv1.PodStatus{
							Reason: "Error",
						},
					},
				},
			},
			kubeEvents: map[string][]k8sv1.Event{
				"ns1": {
					{
						Type: "Normal",
					},
				},
			},
			expectedReport: []*ReportItem{
				{
					Name:     "Pods",
					Problems: []string{fmt.Sprintf("  Pod %s:  \n    Status: %s  \n    Created: %s ago\n", ansi.Color("ErrorPod", "white+b"), ansi.Color("Error", "red+b"), ansi.Color("2s", "white+b"))},
				},
			},
		},
		{
			name:           "Error in replicasets",
			wait:           true,
			kubeNamespaces: []string{"ns1"},
			kubeReplicasets: map[string][]appsv1.ReplicaSet{
				"ns1": {
					{
						Spec: appsv1.ReplicaSetSpec{
							Replicas: ptr.Int32(4),
						},
						Status: appsv1.ReplicaSetStatus{
							Replicas: int32(3),
						},
					},
				},
			},
		},
		{
			name:           "Error in statefulsets",
			wait:           true,
			kubeNamespaces: []string{"ns1"},
			kubeStatefulsets: map[string][]appsv1.StatefulSet{
				"ns1": {
					{
						Spec: appsv1.StatefulSetSpec{
							Replicas: ptr.Int32(4),
						},
						Status: appsv1.StatefulSetStatus{
							Replicas: int32(3),
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		kubeClient := &fakekube.Client{
			Client: fake.NewSimpleClientset(),
		}
		for _, namespace := range testCase.kubeNamespaces {
			_, _ = kubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), &k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}, metav1.CreateOptions{})
		}
		for namespace, podList := range testCase.kubePods {
			for _, pod := range podList {
				pod.ObjectMeta.CreationTimestamp.Time = time.Now()
				_, _ = kubeClient.Client.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
			}
		}
		for namespace, replicasetList := range testCase.kubeReplicasets {
			for _, replicaset := range replicasetList {
				_, _ = kubeClient.Client.AppsV1().ReplicaSets(namespace).Create(context.TODO(), &replicaset, metav1.CreateOptions{})
			}
		}
		for namespace, statefulsetList := range testCase.kubeStatefulsets {
			for _, statefulset := range statefulsetList {
				_, _ = kubeClient.Client.AppsV1().StatefulSets(namespace).Create(context.TODO(), &statefulset, metav1.CreateOptions{})
			}
		}
		for namespace, eventList := range testCase.kubeEvents {
			for _, pod := range eventList {
				_, _ = kubeClient.Client.CoreV1().Events(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
			}
		}

		analyzer := NewAnalyzer(kubeClient, log.Discard)

		report, err := analyzer.CreateReport(testCase.namespace, Options{Wait: testCase.wait})

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		reportAsYaml, err := yaml.Marshal(report)
		assert.NilError(t, err, "Error marshaling report in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedReport)
		assert.NilError(t, err, "Error marshaling expected report in testCase %s", testCase.name)
		assert.Equal(t, string(reportAsYaml), string(expectationAsYaml), "Unexpected report in testCase %s", testCase.name)
	}

}

type reportToStringTestCase struct {
	name string

	report []*ReportItem

	expectedString string
}

func TestReportToString(t *testing.T) {
	testCases := []reportToStringTestCase{
		{
			name:           "No items",
			expectedString: fmt.Sprintf("\n%sNo problems found.\n%sRun `%s` if you want show pod logs\n\n", paddingLeft, paddingLeft, ansi.Color("devspace logs --pick", "white+b")),
		},
		{
			name: "testReport",
			report: []*ReportItem{
				{
					Name: "testReport",
					Problems: []string{
						"Somethings wrong, I guess...",
					},
				},
			},
			expectedString: `
` + ansi.Color(`  ================================================================================
                         testReport (1 potential issue(s))                        
  ================================================================================
`, "green+b") + "Somethings wrong, I guess...\n",
		},
	}

	for _, testCase := range testCases {
		result := ReportToString(testCase.report)
		assert.Equal(t, result, testCase.expectedString, "Unexpected result in testCase %s", testCase.name)
	}
}
