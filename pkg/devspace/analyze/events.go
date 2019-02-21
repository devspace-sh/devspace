package analyze

import (
	"fmt"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// EventRelevanceTime is the time in which events are relevant for us
const EventRelevanceTime = 600 * time.Second

// Events checks the namespace events for warnings
func Events(client *kubernetes.Clientset, config *rest.Config, namespace string) ([]string, error) {
	problems := []string{}

	log.StartWait("Analyzing events")
	defer log.StopWait()

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Get all events
	events, err := client.Core().Events(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// get current time
	now := time.Now()

	// loop through events
	if events.Items != nil {
		for _, event := range events.Items {
			if event.Type != "Normal" && now.Sub(event.LastTimestamp.UTC()) < EventRelevanceTime {
				// This is a bad guess bad usually works
				multiple, _ := meta.UnsafeGuessKindToResource(event.InvolvedObject.GroupVersionKind())

				_, err = dynamicClient.Resource(multiple).Namespace(namespace).Get(event.InvolvedObject.Name, metav1.GetOptions{})
				if err == nil {
					header := ansi.Color(fmt.Sprintf("%s - %s %s: ", event.Type, event.InvolvedObject.Kind, event.InvolvedObject.Name), "202+b")
					problems = append(problems, fmt.Sprintf("%s\n    %dx %s \n", header, int(event.Count), event.Message))
				}
			}
		}
	}

	return problems, nil
}
