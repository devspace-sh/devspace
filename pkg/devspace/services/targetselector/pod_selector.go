package targetselector

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SelectPod let's the user select a pod if necessary and optionally a container
func SelectPod(client kubectl.Client, namespace string, labelSelector *string, question *string, onlyRunning bool, log log.Logger) (*v1.Pod, error) {
	if question == nil {
		question = ptr.String(DefaultPodQuestion)
	}

	if labelSelector != nil {
		podList, err := client.KubeClient().CoreV1().Pods(namespace).List(metav1.ListOptions{
			LabelSelector: *labelSelector,
		})
		if err != nil {
			return nil, err
		}

		if podList.Items != nil && len(podList.Items) == 1 {
			podStatus := kubectl.GetPodStatus(&podList.Items[0])
			if onlyRunning && podStatus != "Running" {
				return nil, nil
			}

			return &podList.Items[0], nil
		} else if podList.Items != nil && len(podList.Items) > 1 {
			options := []string{}
			for _, pod := range podList.Items {
				podStatus := kubectl.GetPodStatus(&pod)
				if onlyRunning && podStatus != "Running" {
					continue
				}

				options = append(options, pod.Name)
			}

			podName := ""
			if len(options) > 1 {
				podName, err = log.Question(&survey.QuestionOptions{
					Question: *question,
					Options:  options,
				})
				if err != nil {
					return nil, err
				}
			} else if len(options) == 1 {
				podName = options[0]
			} else {
				return nil, nil
			}

			for _, pod := range podList.Items {
				if pod.Name == podName {
					return &pod, nil
				}
			}
		}
	}

	podList, err := client.KubeClient().CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if podList.Items != nil && len(podList.Items) == 1 {
		podStatus := kubectl.GetPodStatus(&podList.Items[0])
		if onlyRunning && podStatus != "Running" {
			return nil, nil
		}

		return &podList.Items[0], nil
	} else if podList.Items != nil && len(podList.Items) > 1 {
		options := []string{}
		for _, pod := range podList.Items {
			podStatus := kubectl.GetPodStatus(&pod)
			if onlyRunning && podStatus != "Running" {
				continue
			}

			options = append(options, pod.Name)
		}

		podName := ""
		if len(options) > 1 {
			podName, err = log.Question(&survey.QuestionOptions{
				Question: *question,
				Options:  options,
			})
			if err != nil {
				return nil, err
			}
		} else if len(options) == 1 {
			podName = options[0]
		} else {
			return nil, nil
		}

		for _, pod := range podList.Items {
			if pod.Name == podName {
				return &pod, nil
			}
		}
	}

	return nil, nil
}
