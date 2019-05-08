package helm

import (
	"fmt"
	"strconv"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// WaitForReleasePodToGetReady waits for the release pod to get ready
func WaitForReleasePodToGetReady(client *kubernetes.Clientset, releaseName, releaseNamespace string, releaseRevision int) (*k8sv1.Pod, error) {
	for true {
		time.Sleep(4 * time.Second)

		podList, err := client.Core().Pods(releaseNamespace).List(metav1.ListOptions{
			LabelSelector: "release=" + releaseName,
		})

		if err != nil {
			log.Panicf("Unable to list devspace pods: %s", err.Error())
		}

		if len(podList.Items) > 0 {
			highestRevision := 0
			var selectedPod *k8sv1.Pod

			for i, pod := range podList.Items {
				if kubectl.GetPodStatus(&pod) == "Terminating" {
					continue
				}

				podRevision, podHasRevision := pod.Annotations["revision"]
				hasHigherRevision := (i == 0)

				if !hasHigherRevision && podHasRevision {
					podRevisionInt, _ := strconv.Atoi(podRevision)

					if podRevisionInt > highestRevision {
						hasHigherRevision = true
					}
				}

				if hasHigherRevision {
					selectedPod = &pod
					highestRevision, _ = strconv.Atoi(podRevision)
				}
			}

			if selectedPod != nil {
				_, hasRevision := selectedPod.Annotations["revision"]

				if !hasRevision || highestRevision == releaseRevision {
					if !hasRevision {
						log.Warn("Found pod without revision. Use annotation 'revision' for your pods to avoid this warning.")
					}

					err = waitForPodReady(client, selectedPod, 2*60*time.Second, 5*time.Second)
					if err != nil {
						return nil, fmt.Errorf("Error during waiting for pod: %s", err.Error())
					}

					return selectedPod, nil
				}

				log.Info("Waiting for release upgrade to complete.")
			}
		} else {
			log.Info("Waiting for release to be deployed.")
		}
	}

	return nil, nil
}

func waitForPodReady(kubectl *kubernetes.Clientset, pod *k8sv1.Pod, maxWaitTime time.Duration, checkInterval time.Duration) error {
	for maxWaitTime > 0 {
		pod, err := kubectl.Core().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})

		if err != nil {
			return err
		}

		if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
			return nil
		}

		time.Sleep(checkInterval)
		maxWaitTime = maxWaitTime - checkInterval
	}

	return fmt.Errorf("Max wait time expired")
}
