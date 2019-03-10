package analyze

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ReplicaSets checks replica sets for problems
func ReplicaSets(client *kubernetes.Clientset, namespace string) ([]string, error) {
	problems := []string{}

	log.StartWait("Analyzing replica sets")
	defer log.StopWait()

	// Get all pods
	replicaSets, err := client.Apps().ReplicaSets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Check for issues
	for _, replicaSet := range replicaSets.Items {
		if replicaSet.Status.Replicas != replicaSet.Status.ReadyReplicas {
			problems = append(problems, fmt.Sprintf("ReplicaSet %s replicas are not ready: %d desired, %d ready", replicaSet.Name, replicaSet.Status.Replicas, replicaSet.Status.ReadyReplicas))
		}
		if replicaSet.Status.Replicas != replicaSet.Status.AvailableReplicas {
			problems = append(problems, fmt.Sprintf("ReplicaSet %s replicas are not available: %d desired, %d available", replicaSet.Name, replicaSet.Status.Replicas, replicaSet.Status.AvailableReplicas))
		}
	}

	return problems, nil
}
