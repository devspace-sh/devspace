package services

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SelectContainer let's the user select a container if necessary
func SelectContainer(client *kubernetes.Clientset, namespace string, labelSelector *string, preferContainer *string) (*v1.Pod, *v1.Container, error) {
	pod, err := SelectPod(client, namespace, labelSelector)
	if err != nil {
		return nil, nil, err
	}

	// Select container if necessary
	if pod.Spec.Containers != nil && len(pod.Spec.Containers) == 1 {
		return pod, &pod.Spec.Containers[0], nil
	} else if pod.Spec.Containers != nil && len(pod.Spec.Containers) > 1 {
		options := []string{}
		for _, container := range pod.Spec.Containers {
			if preferContainer != nil && container.Name == *preferContainer {
				return pod, &container, nil
			}

			options = append(options, container.Name)
		}

		containerName := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question: "Select a container",
			Options:  options,
		})
		for _, container := range pod.Spec.Containers {
			if container.Name == containerName {
				return pod, &container, nil
			}
		}
	}

	return nil, nil, nil
}

// SelectPod let's the user select a pod if necessary and optionally a container
func SelectPod(client *kubernetes.Clientset, namespace string, labelSelector *string) (*v1.Pod, error) {
	if labelSelector != nil {
		podList, err := client.Core().Pods(namespace).List(metav1.ListOptions{
			LabelSelector: *labelSelector,
		})
		if err != nil {
			return nil, err
		}

		if podList.Items != nil && len(podList.Items) == 1 {
			return &podList.Items[0], nil
		} else if podList.Items != nil && len(podList.Items) > 1 {
			options := []string{}
			for _, pod := range podList.Items {
				podStatus := kubectl.GetPodStatus(&pod)
				if podStatus != "Running" {
					continue
				}

				options = append(options, pod.Name)
			}

			podName := ""
			if len(options) > 1 {
				podName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
					Question: "Select a pod",
					Options:  options,
				})
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

	podList, err := client.Core().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if podList.Items != nil && len(podList.Items) == 1 {
		return &podList.Items[0], nil
	} else if podList.Items != nil && len(podList.Items) > 1 {
		options := []string{}
		for _, pod := range podList.Items {
			podStatus := kubectl.GetPodStatus(&pod)
			if podStatus != "Running" {
				continue
			}

			options = append(options, pod.Name)
		}

		podName := ""
		if len(options) > 1 {
			podName = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question: "Select a pod",
				Options:  options,
			})
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
