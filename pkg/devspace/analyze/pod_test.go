package analyze

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"gotest.tools/assert"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type podTestCase struct {
	name string

	wait bool
	pod  k8sv1.Pod

	updatedPod *k8sv1.Pod

	expectedProblems []string
	expectedErr      string
}

func TestPods(t *testing.T) {
	testCases := []podTestCase{
		{
			name: "Wait for pod in creation",
			wait: true,
			pod: k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testPod",
				},
				Status: k8sv1.PodStatus{
					Reason: kubectl.WaitStatus[0],
				},
			},
			updatedPod: &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testPod",
				},
				Status: k8sv1.PodStatus{
					Reason:    "Running",
					StartTime: &metav1.Time{Time: time.Now().Add(-MinimumPodAge * 2)},
				},
			},
		},
		{
			name: "Wait for pod in initialization",
			wait: true,
			pod: k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testPod",
				},
				Status: k8sv1.PodStatus{
					Reason: "Init: something",
				},
			},
			updatedPod: &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testPod",
				},
				Status: k8sv1.PodStatus{
					Reason:    "Running",
					StartTime: &metav1.Time{Time: time.Now().Add(-MinimumPodAge * 2)},
				},
			},
		},
		{
			name: "Wait for minimalPodAge to pass",
			wait: true,
			pod: k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testPod",
				},
				Status: k8sv1.PodStatus{
					Reason:    "Running",
					StartTime: &metav1.Time{Time: time.Now()},
				},
			},
			updatedPod: &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testPod",
				},
				Status: k8sv1.PodStatus{
					Reason:    "Running",
					StartTime: &metav1.Time{Time: time.Now().Add(-MinimumPodAge * 2)},
				},
			},
		},
		{
			name: "Analyze pod with many problems",
			wait: false,
			pod: k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testPod",
				},
				Status: k8sv1.PodStatus{
					Reason: "Error",
					ContainerStatuses: []k8sv1.ContainerStatus{
						{
							Ready:        true,
							RestartCount: 1,
							LastTerminationState: k8sv1.ContainerState{
								Terminated: &k8sv1.ContainerStateTerminated{
									FinishedAt: metav1.Time{Time: time.Now().Add(-IgnoreRestartsSince * 2)},
									ExitCode:   int32(1),
									Message:    "someMessage",
									Reason:     "someReason",
								},
							},
						},
						{
							State: k8sv1.ContainerState{
								Terminated: &k8sv1.ContainerStateTerminated{
									FinishedAt: metav1.Time{Time: time.Now().Add(-IgnoreRestartsSince * 2)},
									Message:    "someMessage2",
									Reason:     "someReason2",
									ExitCode:   int32(2),
								},
							},
						},
					},
					InitContainerStatuses: []k8sv1.ContainerStatus{
						{
							Ready: true,
						},
						{
							State: k8sv1.ContainerState{
								Waiting: &k8sv1.ContainerStateWaiting{
									Message: "someMessage3",
									Reason:  "someReason3",
								},
							},
						},
					},
				},
			},
			expectedProblems: []string{
				fmt.Sprintf("Pod %s:", ansi.Color("testPod", "white+b")),
				fmt.Sprintf("    Status: %s", ansi.Color("Init:0/0", "yellow+b")),
				fmt.Sprintf("    Container: %s/2 running", ansi.Color("1", "red+b")),
				"    Problems: ",
				fmt.Sprintf("      - Container: %s", ansi.Color("", "white+b")),
				fmt.Sprintf("        Status: %s (reason: %s)", ansi.Color("Terminated", "red+b"), ansi.Color("someReason2", "red+b")),
				fmt.Sprintf("        Message: %s", ansi.Color("someMessage2", "white+b")),
				fmt.Sprintf("        Last Execution Log: \n%s", ansi.Color("ContainerLogs", "red")),
				"    InitContainer Problems: ",
				fmt.Sprintf("      - Container: %s", ansi.Color("", "white+b")),
				fmt.Sprintf("        Status: %s (reason: %s)", ansi.Color("Waiting", "red+b"), ansi.Color("someReason3", "red+b")),
				fmt.Sprintf("        Message: %s", ansi.Color("someMessage3", "white+b")),
			},
		},
	}

	for _, testCase := range testCases {
		namespace := "testns"
		kubeClient := &fakekube.Client{
			Client: fake.NewSimpleClientset(),
		}
		_, _ = kubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}, metav1.CreateOptions{})
		_, _ = kubeClient.Client.CoreV1().Pods(namespace).Create(context.TODO(), &testCase.pod, metav1.CreateOptions{})

		analyzer := &analyzer{
			client: kubeClient,
			log:    log.Discard,
		}

		go func() {
			time.Sleep(time.Second / 2)
			if testCase.updatedPod != nil {
				_, _ = kubeClient.Client.CoreV1().Pods(namespace).Update(context.TODO(), testCase.updatedPod, metav1.UpdateOptions{})
			}
		}()

		problems, err := analyzer.pods(namespace, Options{Wait: testCase.wait})

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		lineWithTimestamp := regexp.MustCompile("(?m)[\r\n]+^.*ago.*$")
		result := ""
		if len(problems) > 0 {
			result = lineWithTimestamp.ReplaceAllString(problems[0], "")
		}
		expectedString := ""
		if len(testCase.expectedProblems) > 0 {
			expectedString = paddingLeft + strings.Join(testCase.expectedProblems, paddingLeft+"\n") + "\n"
		}
		assert.Equal(t, result, expectedString, "Unexpected problem list in testCase %s", testCase.name)
	}
}
