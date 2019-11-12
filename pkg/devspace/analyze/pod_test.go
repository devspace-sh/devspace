package analyze

/*import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"gotest.tools/assert"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPods(t *testing.T) {
	kubeClient := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testNS",
		},
	})
	assert.NilError(t, err, "Error creating namespace")
	_, err = kubeClient.Client.CoreV1().Pods("testNS").Create(&k8sv1.Pod{
		Status: k8sv1.PodStatus{
			Reason: "Error",
		},
	})
	assert.NilError(t, err, "Error creating pod")

	problems, err := Pods(kubeClient, "testNS", false)
	assert.NilError(t, err, "Error analyzing Pods")
	assert.Equal(t, 1, len(problems), "No problem found with one not running pod")
	assert.Equal(t, true, strings.Contains(problems[0], "Pod"), "Report does not address pods")
	assert.Equal(t, true, strings.Contains(problems[0], "Error"), "Report does not address the pod status")

	timeNow := time.Now()
	_, err = kubeClient.Client.CoreV1().Pods("testNS").Update(&k8sv1.Pod{
		Status: k8sv1.PodStatus{
			Reason: "Running",
			ContainerStatuses: []k8sv1.ContainerStatus{
				k8sv1.ContainerStatus{
					RestartCount: 1,
					LastTerminationState: k8sv1.ContainerState{
						Terminated: &k8sv1.ContainerStateTerminated{
							FinishedAt: metav1.Time{Time: timeNow},
							ExitCode:   1,
							Message:    "This container terminated. Happy debugging!",
							Reason:     "Stopped",
						},
					},
					Ready: false,
					State: k8sv1.ContainerState{
						Waiting: &k8sv1.ContainerStateWaiting{
							Reason:  "Restarting",
							Message: "Restarting after this container hit an error.",
						},
					},
				},
			},
		},
	})
	assert.NilError(t, err, "Error updating pod")

	expectedPodProblem := &podProblem{
		Status:         "Restarting",
		ContainerTotal: 1,
		ContainerProblems: []*containerProblem{
			&containerProblem{
				Name:           "",
				Waiting:        true,
				Reason:         "Restarting",
				Message:        "Restarting after this container hit an error.",
				Restarts:       1,
				LastRestart:    time.Since(timeNow),
				LastExitReason: "Stopped",
				LastExitCode:   1,
				LastMessage:    "This container terminated. Happy debugging!",
			},
		},
	}

	problems, err = Pods(kubeClient, "testNS", false)
	assert.NilError(t, err, "Error analyzing Pods")
	assert.Equal(t, 1, len(problems), "No problem found with one pod with failing containers")

	expectedMessage := printPodProblem(expectedPodProblem)
	re := regexp.MustCompile("(?m)[\r\n]+^.*Created.*$")
	expectedMessage = re.ReplaceAllString(expectedMessage, "")
	problems[0] = re.ReplaceAllString(problems[0], "")
	re = regexp.MustCompile("(?m)[\r\n]+^.*Last Restart.*$")
	expectedMessage = re.ReplaceAllString(expectedMessage, "")
	problems[0] = re.ReplaceAllString(problems[0], "")
	assert.Equal(t, expectedMessage, problems[0], "Wrong pod problem report")
}*/
