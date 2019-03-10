package analyze

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// StatefulSets checks stateful sets for problems
func StatefulSets(client *kubernetes.Clientset, namespace string) ([]string, error) {
	problems := []string{}

	log.StartWait("Analyzing stateful sets")
	defer log.StopWait()

	// Get all pods
	statefulSets, err := client.Apps().StatefulSets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Check for issues
	for _, statefulSet := range statefulSets.Items {
		if statefulSet.Status.Replicas != statefulSet.Status.ReadyReplicas {
			problems = append(problems, fmt.Sprintf("StatefulSet %s desired replicas do not match ready replicas: %d desired, %d ready", statefulSet.Name, statefulSet.Status.Replicas, statefulSet.Status.ReadyReplicas))
		}
		if statefulSet.Status.Replicas != statefulSet.Status.CurrentReplicas {
			problems = append(problems, fmt.Sprintf("StatefulSet %s desired replicas do not match current replicas: %d desired, %d current", statefulSet.Name, statefulSet.Status.Replicas, statefulSet.Status.CurrentReplicas))
		}
	}

	return problems, nil
}
