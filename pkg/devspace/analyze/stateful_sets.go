package analyze

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatefulSets checks stateful sets for problems
func (a *analyzer) statefulSets(namespace string) ([]string, error) {
	problems := []string{}

	// Get all pods
	statefulSets, err := a.client.KubeClient().AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Check for issues
	for _, statefulSet := range statefulSets.Items {
		if statefulSet.Spec.Replicas == nil {
			continue
		}

		desiredReplicas := *statefulSet.Spec.Replicas
		if desiredReplicas != statefulSet.Status.Replicas {
			problems = append(problems, fmt.Sprintf("statefulSet %s desired replicas do not match replicas: %d desired, %d replicas", statefulSet.Name, desiredReplicas, statefulSet.Status.Replicas))
		}
		if desiredReplicas != statefulSet.Status.ReadyReplicas {
			problems = append(problems, fmt.Sprintf("StatefulSet %s desired replicas do not match ready replicas: %d desired, %d ready", statefulSet.Name, desiredReplicas, statefulSet.Status.ReadyReplicas))
		}
		if desiredReplicas != statefulSet.Status.CurrentReplicas {
			problems = append(problems, fmt.Sprintf("StatefulSet %s desired replicas do not match current replicas: %d desired, %d current", statefulSet.Name, desiredReplicas, statefulSet.Status.CurrentReplicas))
		}
	}

	return problems, nil
}
