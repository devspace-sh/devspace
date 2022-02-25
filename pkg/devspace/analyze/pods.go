package analyze

import (
	"context"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MinimumPodAge is the minimum amount of time that a pod should be old
const MinimumPodAge = 10 * time.Second

// WaitTimeout is the amount of time to wait for a pod to start
const WaitTimeout = 120 * time.Second

// IgnoreRestartsSince if they happened 2 hours or later ago
const IgnoreRestartsSince = time.Hour * 2

// Pods analyzes the pods for problems
func (a *analyzer) pods(namespace string, options Options) ([]string, error) {
	var (
		now      = time.Now()
		timeout  = WaitTimeout
		pods     *v1.PodList
		problems = []string{}
		err      error
	)
	if options.Timeout > 0 {
		timeout = time.Duration(options.Timeout) * time.Second
	}

	// Waiting for pods to become active
	if options.Wait {
		for loop := true; loop && time.Since(now) < timeout; {
			loop = false

			// Get all pods
			pods, err = a.client.KubeClient().CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return nil, err
			}

			if pods.Items != nil {
				for _, pod := range pods.Items {
					podStatus := kubectl.GetPodStatus(&pod)
					for _, status := range kubectl.WaitStatus {
						if podStatus == status {
							loop = true
							break
						}
					}

					if strings.HasPrefix(podStatus, "Init:") {
						loop = true
						break
					}

					if podStatus == "Running" && time.Since(pod.Status.StartTime.UTC()) < MinimumPodAge {
						loop = true
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
		pods, err = a.client.KubeClient().CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
	}

	// Analyzing pods
	if pods.Items != nil {
		for _, pod := range pods.Items {
			problem := checkPod(a.client, &pod, options.IgnorePodRestarts)
			if problem != nil {
				problems = append(problems, printPodProblem(problem))
			}
		}
	}

	return problems, nil
}

type podProblem struct {
	Name   string
	Status string

	ContainerReady int
	ContainerTotal int

	InitContainerReady int
	InitContainerTotal int

	Age string

	ContainerProblems     []*containerProblem
	InitContainerProblems []*containerProblem
}

type containerProblem struct {
	Name string

	Restarts    int
	LastRestart time.Duration

	Ready bool

	Terminated   bool
	TerminatedAt time.Duration

	Waiting bool

	Reason  string
	Message string

	LastExitReason         string
	LastExitCode           int
	LastMessage            string
	LastFaultyExecutionLog string
}

// Pod analyzes the pod for potential problems
func checkPod(client kubectl.Client, pod *v1.Pod, ignoreContainerRestarts bool) *podProblem {
	hasProblem := false
	podProblem := &podProblem{
		Name:                  pod.Name,
		Status:                kubectl.GetPodStatus(pod),
		Age:                   time.Since(pod.CreationTimestamp.Time).Round(time.Second).String(),
		ContainerProblems:     []*containerProblem{},
		InitContainerProblems: []*containerProblem{},
	}

	// Check for unusual status
	if _, ok := kubectl.OkayStatus[podProblem.Status]; !ok {
		if !strings.HasPrefix(podProblem.Status, "Init") {
			hasProblem = true
		}
	}

	// Analyze container status
	if pod.Status.ContainerStatuses != nil {
		podProblem.ContainerTotal = len(pod.Status.ContainerStatuses)

		for _, containerStatus := range pod.Status.ContainerStatuses {
			containerProblem := getContainerProblem(client, pod, &containerStatus, ignoreContainerRestarts)
			if containerProblem != nil {
				hasProblem = true

				podProblem.ContainerProblems = append(podProblem.ContainerProblems, containerProblem)
			}

			if containerStatus.Ready {
				podProblem.ContainerReady++
			}
		}
	}

	// Analyze init container status
	if pod.Status.InitContainerStatuses != nil {
		podProblem.InitContainerTotal = len(pod.Status.ContainerStatuses)

		for _, containerStatus := range pod.Status.InitContainerStatuses {
			containerProblem := getContainerProblem(client, pod, &containerStatus, ignoreContainerRestarts)
			if containerProblem != nil {
				hasProblem = true

				podProblem.InitContainerProblems = append(podProblem.InitContainerProblems, containerProblem)
			}

			if containerStatus.Ready {
				podProblem.InitContainerReady++
			}
		}
	}

	if hasProblem {
		return podProblem
	}

	return nil
}

func getContainerProblem(client kubectl.Client, pod *v1.Pod, containerStatus *v1.ContainerStatus, ignoreContainerRestarts bool) *containerProblem {
	tailLines := int64(50)
	hasProblem := false
	containerProblem := &containerProblem{
		Name:     containerStatus.Name,
		Restarts: int(containerStatus.RestartCount),
		Ready:    true,
	}

	// Check if restarted
	if containerStatus.RestartCount > 0 && !ignoreContainerRestarts {
		if containerStatus.LastTerminationState.Terminated != nil && (time.Since(containerStatus.LastTerminationState.Terminated.FinishedAt.Time) < IgnoreRestartsSince) {
			hasProblem = true

			containerProblem.LastRestart = time.Since(containerStatus.LastTerminationState.Terminated.FinishedAt.Time).Round(time.Second)
			containerProblem.LastExitCode = int(containerStatus.LastTerminationState.Terminated.ExitCode)
			containerProblem.LastMessage = containerStatus.LastTerminationState.Terminated.Message
			containerProblem.LastExitReason = containerStatus.LastTerminationState.Terminated.Reason

			if containerProblem.LastExitCode != 0 {
				containerProblem.LastFaultyExecutionLog, _ = client.ReadLogs(context.TODO(), pod.Namespace, pod.Name, containerStatus.Name, containerProblem.Ready, &tailLines)
			}
		}
	}

	// Check if ready
	if !containerStatus.Ready {
		containerProblem.Ready = false

		if containerStatus.State.Terminated != nil {
			containerProblem.Terminated = true
			containerProblem.TerminatedAt = time.Since(containerStatus.State.Terminated.FinishedAt.Time).Round(time.Second)
			containerProblem.Reason = containerStatus.State.Terminated.Reason
			containerProblem.Message = containerStatus.State.Terminated.Message

			containerProblem.LastExitCode = int(containerStatus.State.Terminated.ExitCode)
			if containerProblem.LastExitCode != 0 {
				hasProblem = true
				containerProblem.LastFaultyExecutionLog, _ = client.ReadLogs(context.TODO(), pod.Namespace, pod.Name, containerStatus.Name, false, &tailLines)
			}
		} else if containerStatus.State.Waiting != nil {
			hasProblem = true
			containerProblem.Waiting = true
			containerProblem.Reason = containerStatus.State.Waiting.Reason
			containerProblem.Message = containerStatus.State.Waiting.Message

			// when containerStatus=Waiting && RestartCount>0, print LastFaultyExecutionLog
			if containerStatus.RestartCount > 0 {
				containerProblem.LastExitCode = int(containerStatus.LastTerminationState.Terminated.ExitCode)
				if containerProblem.LastExitCode != 0 {
					containerProblem.LastFaultyExecutionLog, _ = client.ReadLogs(context.TODO(), pod.Namespace, pod.Name, containerStatus.Name, false, &tailLines)
				}
			}
		}
	}

	if hasProblem {
		return containerProblem
	}

	return nil
}
