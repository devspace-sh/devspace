package analyze

import (
	"fmt"
	"testing"
	"time"

	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/mgutz/ansi"
	"gopkg.in/yaml.v2"
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
		analyzeTestCase{},
	}

	for _, testCase := range testCases {
		kubeClient := &fakekube.Client{
			Client: fake.NewSimpleClientset(),
		}
		analyzer := NewAnalyzer(kubeClient, log.Discard)

		err := analyzer.Analyze(testCase.namespace, testCase.noWait)

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
	noWait    bool

	expectedErr    string
	expectedReport []*ReportItem
}

/*Part of this function is untestable right now because the helper function events uses the RestClient of the KubeClient.
The fake client returns nil. Therefore it's not possible to let events return problems.*/
func TestCreateReport(t *testing.T) {
	testCases := []createReportTestCase{
		createReportTestCase{
			name:           "Nothing to report",
			kubeNamespaces: []string{"ns1"},
			kubeReplicasets: map[string][]appsv1.ReplicaSet{
				"ns1": []appsv1.ReplicaSet{
					appsv1.ReplicaSet{
						Spec: appsv1.ReplicaSetSpec{},
					},
				},
			},
		},
		createReportTestCase{
			name:           "Error in pods",
			kubeNamespaces: []string{"ns1"},
			kubePods: map[string][]k8sv1.Pod{
				"ns1": []k8sv1.Pod{
					k8sv1.Pod{
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
				"ns1": []k8sv1.Event{
					k8sv1.Event{
						Type: "Normal",
					},
				},
			},
			expectedReport: []*ReportItem{
				&ReportItem{
					Name:     "Pods",
					Problems: []string{fmt.Sprintf("  Pod %s:  \n    Status: %s  \n    Created: %s ago\n", ansi.Color("ErrorPod", "white+b"), ansi.Color("Error", "red+b"), ansi.Color("1s", "white+b"))},
				},
			},
		},
		createReportTestCase{
			name:           "Error in replicasets",
			kubeNamespaces: []string{"ns1"},
			kubeReplicasets: map[string][]appsv1.ReplicaSet{
				"ns1": []appsv1.ReplicaSet{
					appsv1.ReplicaSet{
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
		createReportTestCase{
			name:           "Error in statefulsets",
			kubeNamespaces: []string{"ns1"},
			kubeStatefulsets: map[string][]appsv1.StatefulSet{
				"ns1": []appsv1.StatefulSet{
					appsv1.StatefulSet{
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
			kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			})
		}
		for namespace, podList := range testCase.kubePods {
			for _, pod := range podList {
				pod.ObjectMeta.CreationTimestamp.Time = time.Now()
				kubeClient.Client.CoreV1().Pods(namespace).Create(&pod)
			}
		}
		for namespace, replicasetList := range testCase.kubeReplicasets {
			for _, replicaset := range replicasetList {
				kubeClient.Client.AppsV1().ReplicaSets(namespace).Create(&replicaset)
			}
		}
		for namespace, statefulsetList := range testCase.kubeStatefulsets {
			for _, statefulset := range statefulsetList {
				kubeClient.Client.AppsV1().StatefulSets(namespace).Create(&statefulset)
			}
		}
		for namespace, eventList := range testCase.kubeEvents {
			for _, pod := range eventList {
				kubeClient.Client.CoreV1().Events(namespace).Create(&pod)
			}
		}

		analyzer := NewAnalyzer(kubeClient, log.Discard)

		report, err := analyzer.CreateReport(testCase.namespace, testCase.noWait)

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
		reportToStringTestCase{
			name:           "No items",
			expectedString: fmt.Sprintf("\n%sNo problems found.\n%sRun `%s` if you want show pod logs\n\n", paddingLeft, paddingLeft, ansi.Color("devspace logs --pick", "white+b")),
		},
		reportToStringTestCase{
			name: "testReport",
			report: []*ReportItem{
				&ReportItem{
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
