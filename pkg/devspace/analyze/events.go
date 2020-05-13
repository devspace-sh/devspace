package analyze

import (
	"context"
	"fmt"
	"time"

	"github.com/mgutz/ansi"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// EventRelevanceTime is the time in which events are relevant for us
const EventRelevanceTime = 1800 * time.Second

// Events checks the namespace events for warnings
func (a *analyzer) events(namespace string) ([]string, error) {
	problems := []string{}

	// Get all events
	events, err := a.client.KubeClient().CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// loop through events
	if events.Items != nil {
		for _, event := range events.Items {
			if event.Type != "Normal" {
				// This is a bad guess, but works for most resources
				multiple, _ := meta.UnsafeGuessKindToResource(event.InvolvedObject.GroupVersionKind())

				_, err = a.client.KubeClient().CoreV1().RESTClient().Get().AbsPath(makeURLSegments(multiple, namespace, event.InvolvedObject.Name)...).Do(context.TODO()).Get()
				if err == nil {
					header := ansi.Color(fmt.Sprintf("%s (%s ago) - %s %s: ", event.Type, time.Since(event.LastTimestamp.Time).Round(time.Second).String(), event.InvolvedObject.Kind, event.InvolvedObject.Name), "202+b")
					problems = append(problems, paddingLeft+fmt.Sprintf("%s\n%s%dx %s \n", header, paddingLeft, int(event.Count), event.Message))
				}
			}
		}
	}

	return problems, nil
}

// Copied from dynamic client
func makeURLSegments(resource schema.GroupVersionResource, namespace, name string) []string {
	url := []string{}
	if len(resource.Group) == 0 {
		url = append(url, "api")
	} else {
		url = append(url, "apis", resource.Group)
	}
	url = append(url, resource.Version)

	if len(namespace) > 0 {
		url = append(url, "namespaces", namespace)
	}
	url = append(url, resource.Resource)

	if len(name) > 0 {
		url = append(url, name)
	}

	return url
}
