package analyze

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/mgutz/ansi"
)

// Prints the pod problem to a string
func printPodProblem(pp *podProblem) string {
	s := []string{
		fmt.Sprintf("Pod %s:", ansi.Color(pp.Name, "white+b")),
	}

	// Status
	formattedStatus := pp.Status
	if _, ok := kubectl.OkayStatus[pp.Status]; ok {
		formattedStatus = ansi.Color(formattedStatus, "green+b")
	} else if _, ok := kubectl.CriticalStatus[pp.Status]; ok {
		formattedStatus = ansi.Color(formattedStatus, "red+b")
	} else {
		formattedStatus = ansi.Color(formattedStatus, "yellow+b")
	}
	s = append(s, fmt.Sprintf("    Status: %s", formattedStatus))
	s = append(s, fmt.Sprintf("    Created: %s ago", ansi.Color(pp.Age, "white+b")))

	// Container
	if pp.ContainerTotal > 0 {
		readyContainer := strconv.Itoa(pp.ContainerReady)
		if pp.ContainerTotal != pp.ContainerReady {
			readyContainer = ansi.Color(readyContainer, "red+b")
		}

		s = append(s, fmt.Sprintf("    Container: %s/%d running", readyContainer, pp.ContainerTotal))

		// Container problems
		if len(pp.ContainerProblems) > 0 {
			s = append(s, "    Problems: ")

			for _, containerProblem := range pp.ContainerProblems {
				s = append(s, printContainerProblem(containerProblem)...)
			}
		}

		// Init Container problems
		if len(pp.InitContainerProblems) > 0 {
			s = append(s, "    InitContainer Problems: ")

			for _, containerProblem := range pp.InitContainerProblems {
				s = append(s, printContainerProblem(containerProblem)...)
			}
		}
	}

	return paddingLeft + strings.Join(s, paddingLeft+"\n") + "\n"
}

func printContainerProblem(containerProblem *containerProblem) []string {
	s := []string{}
	s = append(s, fmt.Sprintf("      - Container: %s", ansi.Color(containerProblem.Name, "white+b")))

	if containerProblem.Waiting {
		s = append(s, fmt.Sprintf("        Status: %s (reason: %s)", ansi.Color("Waiting", "red+b"), ansi.Color(containerProblem.Reason, "red+b")))
	} else if containerProblem.Terminated {
		s = append(s, fmt.Sprintf("        Status: %s (reason: %s)", ansi.Color("Terminated", "red+b"), ansi.Color(containerProblem.Reason, "red+b")))
		s = append(s, fmt.Sprintf("        Terminated: %s ago", ansi.Color(containerProblem.TerminatedAt.String(), "white+b")))
	}
	if containerProblem.Message != "" {
		s = append(s, fmt.Sprintf("        Message: %s", ansi.Color(containerProblem.Message, "white+b")))
	}

	if containerProblem.Restarts > 0 {
		s = append(s, fmt.Sprintf("        Restarts: %s", ansi.Color(strconv.Itoa(containerProblem.Restarts), "red+b")))
		s = append(s, fmt.Sprintf("        Last Restart: %s ago", ansi.Color(containerProblem.LastRestart.String(), "white+b")))
		if containerProblem.LastExitCode != 0 {
			s = append(s, fmt.Sprintf("        Last Exit: %s (Code: %s)", ansi.Color(containerProblem.LastExitReason, "red+b"), ansi.Color(strconv.Itoa(containerProblem.LastExitCode), "red+b")))
		}
		if containerProblem.LastMessage != "" {
			s = append(s, fmt.Sprintf("        Last Exit Message: %s", ansi.Color(containerProblem.LastMessage, "red+b")))
		}
	}

	if containerProblem.LastFaultyExecutionLog != "" {
		s = append(s, fmt.Sprintf("        Last Execution Log: \n%s", ansi.Color(containerProblem.LastFaultyExecutionLog, "red")))
	}

	return s
}
