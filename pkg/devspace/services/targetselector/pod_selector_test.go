package targetselector

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"gotest.tools/assert"
)

func TestPodSelectionOneNotRunningPodLabelMatches(t *testing.T) {
	namespace := "test"

	//Create namespace
	kubeClient := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}
	_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	//Create one Pod with label that will match
	matchingPodLabels := make(map[string]string, 1)
	matchingPodLabels["DoesItMatch"] = "Yes"
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name:   "MatchingPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Create Pod wit label that does not match
	unmatchingPodLabels := make(map[string]string, 1)
	unmatchingPodLabels["DoesItMatch"] = "No"
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: unmatchingPodLabels,
			Name:   "UnMatchingPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Test SelectPod with labelSelector that matches with one pod
	labelSelector := "DoesItMatch=Yes"
	returnedPod, err := SelectPod(kubeClient, namespace, &labelSelector, nil, true, log.GetInstance())
	if err != nil {
		t.Fatalf("Error selecting pod: %v", err)
	}
	assert.Equal(t, true, returnedPod == nil, "returned Pod is not nil")
}

func TestPodSelectionTwoNotRunningPodsLabelMatches(t *testing.T) {
	namespace := "test"

	//Create namespace
	kubeClient := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}
	_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	//Create two Pods with label that will match
	matchingPodLabels := make(map[string]string, 1)
	matchingPodLabels["DoesItMatch"] = "Yes"
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name:   "MatchingPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name:   "OtherMatchingPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Create Pod wit label that does not match
	unmatchingPodLabels := make(map[string]string, 1)
	unmatchingPodLabels["DoesItMatch"] = "No"
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: unmatchingPodLabels,
			Name:   "UnMatchingPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Test SelectPod with labelSelector that matches with one pod
	labelSelector := "DoesItMatch=Yes"
	returnedPod, err := SelectPod(kubeClient, namespace, &labelSelector, nil, true, log.GetInstance())
	if err != nil {
		t.Fatalf("Error selecting pod: %v", err)
	}
	assert.Equal(t, true, returnedPod == nil, "returned Pod is not nil")
}

func TestPodSelectionOneRunningOneNotRunningPodsLabelMatches(t *testing.T) {
	namespace := "test"

	//Create namespace
	kubeClient := &kubectl.Client{Client: fake.NewSimpleClientset()}
	_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	//Create two Pods with label that will match
	matchingPodLabels := make(map[string]string, 1)
	matchingPodLabels["DoesItMatch"] = "Yes"
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name:   "RunningMatchingPod",
		},
		Status: k8sv1.PodStatus{
			Reason: "Running",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name:   "StoppedMatchingPod",
		},
		Status: k8sv1.PodStatus{
			Reason: "Stopped",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Create Pod wit label that does not match
	unmatchingPodLabels := make(map[string]string, 1)
	unmatchingPodLabels["DoesItMatch"] = "No"
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: unmatchingPodLabels,
			Name:   "UnMatchingPod",
		},
		Status: k8sv1.PodStatus{
			Reason: "Running",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Test SelectPod with labelSelector that matches with one pod
	labelSelector := "DoesItMatch=Yes"
	returnedPod, err := SelectPod(kubeClient, namespace, &labelSelector, nil, true, log.GetInstance())
	if err != nil {
		t.Fatalf("Error selecting pod: %v", err)
	}
	assert.Equal(t, false, returnedPod == nil, "returned Pod is nil")
	assert.Equal(t, returnedPod.Labels["DoesItMatch"], "Yes", "SelectPod returned the wrong pod")
	assert.Equal(t, returnedPod.Name == "RunningMatchingPod", true, "SelectPod returned the wrong pod")
}

func TestPodSelectionOneNotRunningPodNoLabel(t *testing.T) {
	namespace := "test"

	//Create namespace
	kubeClient := &kubectl.Client{Client: fake.NewSimpleClientset()}
	_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	//Create one Pod with label that will match
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "StoppedPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Test SelectPod
	returnedPod, err := SelectPod(kubeClient, namespace, nil, nil, true, log.GetInstance())
	if err != nil {
		t.Fatalf("Error selecting pod: %v", err)
	}
	assert.Equal(t, true, returnedPod == nil, "returned Pod is not nil")
}

func TestPodSelectionTwoNotRunningPodsNoLabel(t *testing.T) {
	namespace := "test"

	//Create namespace
	kubeClient := &kubectl.Client{Client: fake.NewSimpleClientset()}
	_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	//Create two Pods
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "StoppedPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "OtherStoppedPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Test SelectPod
	returnedPod, err := SelectPod(kubeClient, namespace, nil, nil, true, log.GetInstance())
	if err != nil {
		t.Fatalf("Error selecting pod: %v", err)
	}
	assert.Equal(t, true, returnedPod == nil, "returned Pod is not nil")
}

func TestPodSelectionOneRunningOneNotRunningPodsNoLabel(t *testing.T) {
	namespace := "test"

	//Create namespace
	kubeClient := &kubectl.Client{Client: fake.NewSimpleClientset()}
	_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	//Create two Pods
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "RunningPod",
		},
		Status: k8sv1.PodStatus{
			Reason: "Running",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "StoppedPod",
		},
		Status: k8sv1.PodStatus{
			Reason: "Stopped",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Test SelectPod
	returnedPod, err := SelectPod(kubeClient, namespace, nil, nil, true, log.GetInstance())
	if err != nil {
		t.Fatalf("%s: %v", message.SelectorErrorPod, err)
	}
	assert.Equal(t, false, returnedPod == nil, "returned Pod is nil")
	assert.Equal(t, returnedPod.Name == "RunningPod", true, "SelectPod returned the wrong pod")
}

/*func TestPodSelection(t *testing.T) {
	namespace := "test"

	//Create namespace
	kubeClient := &kubectl.Client{ 		Client: fake.NewSimpleClientset(), 	}
	_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	//Create two Pods with label that will match
	matchingPodLabels := make(map[string]string, 1)
	matchingPodLabels["DoesItMatch"] = "Yes"
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name: "MatchingPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name: "OtherMatchingPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Create Pod wit label that does not match
	unmatchingPodLabels := make(map[string]string, 1)
	unmatchingPodLabels["DoesItMatch"] = "No"
	_, err = kubeClient.Client.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: unmatchingPodLabels,
			Name: "UnMatchingPod",
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//Test SelectPod with labelSelector that matches with one pod
	labelSelector := "DoesItMatch=Yes"
	returnedPod, err := SelectPod(kubeClient, namespace, &labelSelector)
	if err != nil {
		t.Fatalf("Error selecting pod: %v", err)
	}
	assert.Equal(t, false, returnedPod == nil, "returned Pod is nil")
	assert.Equal(t, returnedPod.Labels["DoesItMatch"], "Yes", "SelectPod returned the wrong pod")
	assert.Equal(t, returnedPod.Name == "MatchingPod" || returnedPod.Name == "OtherMatchingPod", true, "SelectPod returned the wrong pod")

	//Delete othermatching pod and try again
	err = kubeClient.Client.CoreV1().Pods(namespace).Delete("OtherMatchingPod", &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error deleting pod: %v", err)
	}
	returnedPod, err = SelectPod(kubeClient, namespace, &labelSelector)
	if err != nil {
		t.Fatalf("Error selecting pod: %v", err)
	}
	assert.Equal(t, false, returnedPod == nil, "returned Pod with labelSelector is nil")
	assert.Equal(t, returnedPod.Labels["DoesItMatch"], "No", "SelectPod returned deleted pod")
	assert.Equal(t, returnedPod.Name, "MatchingPod", "SelectPod returned deleted pod")

	//Delete matching pod and try again
	err = kubeClient.Client.CoreV1().Pods(namespace).Delete("OtherMatchingPod", &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error deleting pod: %v", err)
	}
	returnedPod, err = SelectPod(kubeClient, namespace, &labelSelector)
	if err != nil {
		t.Fatalf("Error selecting pod: %v", err)
	}
	assert.Equal(t, false, returnedPod == nil, "returned Pod with labelSelector is nil")
	assert.Equal(t, returnedPod.Labels["DoesItMatch"], "No", "SelectPod returned deleted pod")
	assert.Equal(t, returnedPod.Name, "UnMatchingPod", "SelectPod returned deleted pod")

}*/
