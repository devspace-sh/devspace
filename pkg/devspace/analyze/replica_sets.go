package analyze

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReplicaSets checks replica sets for problems
func (a *analyzer) replicaSets(namespace string) ([]string, error) {
	problems := []string{}

	// Get all pods
	replicaSets, err := a.client.KubeClient().AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Check for issues
	for _, replicaSet := range replicaSets.Items {
		if replicaSet.Spec.Replicas == nil {
			continue
		}

		desiredReplicas := *replicaSet.Spec.Replicas
		if desiredReplicas != replicaSet.Status.Replicas {
			problems = append(problems, fmt.Sprintf("ReplicaSet %s desired replicas unequal with replicas: %d desired, %d replicas", replicaSet.Name, desiredReplicas, replicaSet.Status.Replicas))
		}
		if desiredReplicas != replicaSet.Status.ReadyReplicas {
			problems = append(problems, fmt.Sprintf("ReplicaSet %s replicas are not ready: %d desired, %d ready", replicaSet.Name, desiredReplicas, replicaSet.Status.ReadyReplicas))
		}
		if desiredReplicas != replicaSet.Status.AvailableReplicas {
			problems = append(problems, fmt.Sprintf("ReplicaSet %s replicas are not available: %d desired, %d available", replicaSet.Name, desiredReplicas, replicaSet.Status.AvailableReplicas))
		}
	}

	return problems, nil
}
