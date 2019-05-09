package targetselector

import (
	"fmt"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	
	"k8s.io/client-go/kubernetes/fake"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	"gotest.tools/assert"
)

func TestTargetSelector(t *testing.T) {
	namespace := "test"
	selectedContainerName := "TestContainer2"
	selectedPodName := "TestPod"
	// @Florian

	//Create targetSelector
	config := latest.Config{
		Cluster: &latest.Cluster{
			Namespace: &namespace,
		},
	}
	selectorParameter := SelectorParameter{
		CmdParameter: CmdParameter{
			ContainerName: &selectedContainerName,
			PodName: &selectedPodName,
		},
	}
	targetSelector, err := NewTargetSelector(&config, &selectorParameter, true)
	if err != nil {
		t.Fatalf("Error creating targetSelector: %v", err)
	}

	//Setting up kubeClient
	kubeClient := fake.NewSimpleClientset()
	_, err = kubeClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}
	matchingPodLabels := make(map[string]string, 1)
	matchingPodLabels["DoesItMatch"] = "Yes"
	_, err = kubeClient.CoreV1().Pods(namespace).Create(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name: selectedPodName,
		},
		Status: k8sv1.PodStatus{
			Reason: "Running",
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				k8sv1.Container{
					Name: "TestContainer",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}

	//First test
	returnedPod, returnedContainer, err := targetSelector.GetContainer(kubeClient)
	if err != nil {
		t.Fatalf("Error getting Container: %v", err)
	}
	assert.Equal(t, false, returnedPod == nil, "returned Pod is nil")
	assert.Equal(t, false, returnedContainer == nil, "returned container is nil")
	assert.Equal(t, returnedPod.Name, selectedPodName, "Wrong pod returned")
	assert.Equal(t, returnedContainer.Name, "TestContainer", "Wrong container returned")

	//The pod stops running
	matchingPodLabels["DoesItMatch"] = "Yes"
	_, err = kubeClient.CoreV1().Pods(namespace).Update(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name: selectedPodName,
		},
		Status: k8sv1.PodStatus{
			Reason: "Stopped",
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				k8sv1.Container{
					Name: "TestContainer",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Error creating pod: %v", err)
	}
	returnedPod, returnedContainer, err = targetSelector.GetContainer(kubeClient)
	assert.Equal(t, false, err == nil, "No error from selecting in an empty namespace")
	assert.Equal(t, fmt.Sprintf("Couldn't get pod %s, because pod has status: %s", selectedPodName, "Stopped"), err.Error(), "Wrong error")
	assert.Equal(t, true, returnedPod == nil, "returned Pod is not nil")
	assert.Equal(t, true, returnedContainer == nil, "returned container is not nil")

	//Now with two containers
	kubeClient.CoreV1().Pods(namespace).Update(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name: "TestPod",
		},
		Status: k8sv1.PodStatus{
			Reason: "Running",
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				k8sv1.Container{
					Name: "TestContainer",
				},
				k8sv1.Container{
					Name: "TestContainer2",
				},
			},
		},
	})
	returnedPod, returnedContainer, err = targetSelector.GetContainer(kubeClient)
	if err != nil {
		t.Fatalf("Error getting Container: %v", err)
	}
	assert.Equal(t, false, returnedPod == nil, "returned Pod is nil")
	assert.Equal(t, false, returnedContainer == nil, "returned container is nil")
	assert.Equal(t, returnedPod.Name, selectedPodName, "Wrong pod returned")
	assert.Equal(t, returnedContainer.Name, selectedContainerName, "Wrong container returned")

	//Still multiple containers but given Containername doesn't exist
	notExistentContainerName := "DoesntExist"
	targetSelector.containerName = &notExistentContainerName
	returnedPod, returnedContainer, err = targetSelector.GetContainer(kubeClient)
	assert.Equal(t, false, err == nil, "No error from selecting in an empty namespace")
	assert.Equal(t, fmt.Sprintf("Couldn't find container %s in pod %s", notExistentContainerName, selectedPodName), err.Error(), "Wrong error")
	assert.Equal(t, true, returnedPod == nil, "returned Pod is not nil")
	assert.Equal(t, true, returnedContainer == nil, "returned container is not nil")
	targetSelector.containerName = &selectedContainerName

	//Now with zero containers
	kubeClient.CoreV1().Pods(namespace).Update(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name: "TestPod",
		},
		Status: k8sv1.PodStatus{
			Reason: "Running",
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{},
		},
	})
	returnedPod, returnedContainer, err = targetSelector.GetContainer(kubeClient)
	if err != nil {
		t.Fatalf("Error getting Container: %v", err)
	}
	assert.Equal(t, false, returnedPod == nil, "returned Pod is nil")
	assert.Equal(t, true, returnedContainer == nil, "returned container is not nil")
	assert.Equal(t, returnedPod.Name, "TestPod", "Wrong pod returned")

	//The pod name doesn't exist
	notExistentPodName := "DoesntExist"
	targetSelector.podName = &notExistentPodName
	returnedPod, returnedContainer, err = targetSelector.GetContainer(kubeClient)
	assert.Equal(t, false, err == nil, "No error from selecting in an empty namespace")
	assert.Equal(t, fmt.Sprintf("pods \"%s\" not found", notExistentPodName), err.Error(), "Wrong error")
	assert.Equal(t, true, returnedPod == nil, "returned Pod is not nil")
	assert.Equal(t, true, returnedContainer == nil, "returned container is not nil")
	targetSelector.podName = &notExistentPodName

	//Now with an empty namespace
	emptyNamespace := "empty"
	containerName := "DoesntMatterAnyway"
	sp := SelectorParameter{
		ConfigParameter: ConfigParameter{
			Namespace: &emptyNamespace,
			ContainerName: &containerName,
		},
	}
	emptyNamespaceTargetSelector, err := NewTargetSelector(&latest.Config{}, &sp, true)
	if err != nil {
		t.Fatalf("Error creating TargetSelector: %v", err)
	}
	returnedPod, returnedContainer, err = emptyNamespaceTargetSelector.GetContainer(kubeClient)
	assert.Equal(t, false, err == nil, "No error from selecting in an empty namespace")
	assert.Equal(t, "Couldn't find a running pod in namespace " + emptyNamespace, err.Error(), "Wrong error")
	assert.Equal(t, true, returnedPod == nil, "returned Pod is not nil")
	assert.Equal(t, true, returnedContainer == nil, "returned container is not nil")

	//Don't allow pick but multiple Containers
	kubeClient.CoreV1().Pods(namespace).Update(&k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: matchingPodLabels,
			Name: "TestPod",
		},
		Status: k8sv1.PodStatus{
			Reason: "Running",
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				k8sv1.Container{
					Name: "TestContainer",
				},
				k8sv1.Container{
					Name: "TestContainer2",
				},
			},
		},
	})
	selectorParameter.CmdParameter.ContainerName = nil
	noPickTargetSelector, err := NewTargetSelector(&config, &selectorParameter, false)
	if err != nil {
		t.Fatalf("Error creating TargetSelector: %v", err)
	}
	returnedPod, returnedContainer, err = noPickTargetSelector.GetContainer(kubeClient)
	assert.Equal(t, false, err == nil, "No error from getting one of multiple containers withou picking")
	assert.Equal(t, fmt.Sprintf("Couldn't select a container in pod %s, because no container name was specified", selectedPodName), err.Error(), "Wrong error")
	assert.Equal(t, true, returnedPod == nil, "returned Pod is not nil")
	assert.Equal(t, true, returnedContainer == nil, "returned container is not nil")
}
