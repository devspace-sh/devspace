package analyze

import (
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// MinimumPodAge is the minimum amount of time that a pod should be old
const MinimumPodAge = 60 * time.Second

// WaitTimeout is the amount of time to wait for a pod to start
const WaitTimeout = 60 * time.Second

// WaitStatus are the status to wait
var WaitStatus = []string{
	"ContainerCreating",
	"Pending",
	"Terminating",
}

// CriticalStatus container status
var CriticalStatus = map[string]string{
	"Error":                      "",
	"Unknown":                    "",
	"ImagePullBackOff":           "",
	"CrashLoopBackOff":           "",
	"RunContainerError":          "",
	"ErrImagePull":               "",
	"CreateContainerConfigError": "",
	"InvalidImageName":           "",
}

// OkayStatus container status
var OkayStatus = map[string]string{
	"Completed": "",
	"Running":   "",
}

// IgnoreRestartsSince if they happened 2 hours or later ago
const IgnoreRestartsSince = time.Hour * 2

// Pods analyzes the pods for problems
func Pods(client *kubernetes.Clientset, namespace string, noWait bool) ([]string, error) {
	problems := []string{}

	log.StartWait("Analyzing pods")
	defer log.StopWait()

	// Get current time
	now := time.Now()

	var pods *v1.PodList
	var err error

	// Waiting for pods to become active
	if noWait == false {
		for loop := true; loop && time.Now().Sub(now) < WaitTimeout; {
			loop = false

			// Get all pods
			pods, err = client.Core().Pods(namespace).List(metav1.ListOptions{})
			if err != nil {
				return nil, err
			}

			if pods.Items != nil {
				for _, pod := range pods.Items {
					podStatus := kubectl.GetPodStatus(&pod)
					if strings.HasPrefix(podStatus, "Init") {
						loop = true
						log.StartWait("Waiting for pod " + pod.Name + " init container startup")
						break
					}

					for _, status := range WaitStatus {
						if podStatus == status {
							loop = true
							log.StartWait("Waiting for pod " + pod.Name + " with status " + podStatus)
							break
						}
					}

					if podStatus == "Running" && now.Sub(pod.Status.StartTime.UTC()) < MinimumPodAge {
						loop = true
						log.StartWait("Waiting for pod " + pod.Name + " startup")
						break
					}

					if loop {
						break
					}
				}
			}

			time.Sleep(time.Second)
		}
	} else {
		// Get all pods
		pods, err = client.Core().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
	}

	// Analyzing pods
	if pods.Items != nil {
		for _, pod := range pods.Items {
			problem := Pod(&pod)
			if problem != "" {
				problems = append(problems, problem)
			}
		}
	}

	return problems, nil
}

// Pod analyzes the pod for potential problems
func Pod(pod *v1.Pod) string {
	problems := []string{}
	padding := newString(" ", 5)
	containerPadding := newString(" ", 3)

	// Analyze container status
	if pod.Status.ContainerStatuses != nil {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			containerProblems := getContainerProblems(&containerStatus)
			if len(containerProblems) > 0 {
				ready := ""
				if containerStatus.Ready == false {
					ready = " (not running)"
				}

				problems = append(problems, fmt.Sprintf("Container %s%s:", ansi.Color(containerStatus.Name, "white+b"), ready))
				for _, cp := range containerProblems {
					problems = append(problems, containerPadding+cp)
				}
			}
		}
	}

	// Analyze init container status
	if pod.Status.InitContainerStatuses != nil {
		for _, containerStatus := range pod.Status.InitContainerStatuses {
			containerProblems := getContainerProblems(&containerStatus)
			if len(containerProblems) > 0 {
				ready := ""
				if containerStatus.Ready == false {
					ready = " (not running)"
				}

				problems = append(problems, fmt.Sprintf("Init Container %s%s:", ansi.Color(containerStatus.Name, "white+b"), ready))
				for _, cp := range containerProblems {
					problems = append(problems, containerPadding+cp)
				}
			}
		}
	}

	if len(problems) > 0 {
		podStatus := kubectl.GetPodStatus(pod)
		header := fmt.Sprintf("Pod %s (%s):", ansi.Color(pod.Name, "white+b"), ansi.Color(podStatus, "red+b"))

		problem := header + "\n"
		for _, p := range problems {
			problem += padding + p + "\n"
		}

		problem += "\n"

		return problem
	}

	return ""
}

func getContainerProblems(containerStatus *v1.ContainerStatus) []string {
	problems := []string{}
	now := time.Now()

	// Check if ready
	if containerStatus.Ready == false {
		if containerStatus.State.Terminated != nil {
			occured := now.Sub(containerStatus.State.Terminated.FinishedAt.Time).Round(time.Second).String()
			reason := ansi.Color(containerStatus.State.Terminated.Reason, "white+b")

			message := fmt.Sprintf("Currently terminated %s ago, with reason %s and exitCode %d", occured, reason, containerStatus.State.Terminated.ExitCode)
			problems = append(problems, message)

			if containerStatus.State.Terminated.Message != "" {
				problems = append(problems, "  Message: "+ansi.Color(containerStatus.State.Terminated.Message, "white"))
			}
		} else if containerStatus.State.Waiting != nil {
			message := fmt.Sprintf("Currently waiting, with reason %s", containerStatus.State.Waiting.Reason)
			problems = append(problems, message)

			if containerStatus.State.Waiting.Message != "" {
				problems = append(problems, "  Message: "+ansi.Color(containerStatus.State.Waiting.Message, "white"))
			}
		}
	}

	// Check if restarted
	if containerStatus.RestartCount > 0 {
		if containerStatus.LastTerminationState.Terminated != nil && now.Sub(containerStatus.LastTerminationState.Terminated.FinishedAt.UTC()) < IgnoreRestartsSince {
			restartCount := ansi.Color(fmt.Sprintf("%d", containerStatus.RestartCount), "white+b")
			reason := ansi.Color(containerStatus.LastTerminationState.Terminated.Reason, "white+b")
			occured := now.Sub(containerStatus.LastTerminationState.Terminated.FinishedAt.Time).Round(time.Second).String()
			exitCode := int(containerStatus.LastTerminationState.Terminated.ExitCode)

			problems = append(problems, fmt.Sprintf("Restarted %s times: last restart (%s) was %s ago with exit code %d", restartCount, reason, occured, exitCode))
			if containerStatus.LastTerminationState.Terminated.Message != "" {
				problems = append(problems, "  Message: "+ansi.Color(containerStatus.LastTerminationState.Terminated.Message, "white"))
			}
		}
	}

	return problems
}
