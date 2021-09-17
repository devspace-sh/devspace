package analyze

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mgutz/ansi"
	"gotest.tools/assert"
)

type printPodProblemTestCase struct {
	name string

	podProblem podProblem

	expectedString []string
}

func TestPrintPodProblem(t *testing.T) {
	testCases := []printPodProblemTestCase{
		{
			name: "Pod with lots of problems",
			podProblem: podProblem{
				Name:           "testPod1",
				Status:         "Running",
				ContainerTotal: 2,
				ContainerReady: 1,
				ContainerProblems: []*containerProblem{
					{
						Name:                   "testContainer1",
						Waiting:                true,
						Reason:                 "testReason1",
						Message:                "TestMessage1",
						Restarts:               1,
						LastExitReason:         "testExitReason1",
						LastExitCode:           1,
						LastMessage:            "ContainerError1",
						LastFaultyExecutionLog: "TestLog1",
					},
				},
				InitContainerProblems: []*containerProblem{
					{
						Name:       "testContainer2",
						Terminated: true,
						Reason:     "testReason2",
					},
				},
			},
			expectedString: []string{
				fmt.Sprintf("Pod %s:", ansi.Color("testPod1", "white+b")),
				fmt.Sprintf("    Status: %s", ansi.Color("Running", "green+b")),
				fmt.Sprintf("    Created: %s ago", ansi.Color("", "white+b")),
				fmt.Sprintf("    Container: %s/2 running", ansi.Color("1", "red+b")),
				"    Problems: ",
				fmt.Sprintf("      - Container: %s", ansi.Color("testContainer1", "white+b")),
				fmt.Sprintf("        Status: %s (reason: %s)", ansi.Color("Waiting", "red+b"), ansi.Color("testReason1", "red+b")),
				fmt.Sprintf("        Message: %s", ansi.Color("TestMessage1", "white+b")),
				fmt.Sprintf("        Restarts: %s", ansi.Color("1", "red+b")),
				fmt.Sprintf("        Last Restart: %s ago", ansi.Color("0s", "white+b")),
				fmt.Sprintf("        Last Exit: %s (Code: %s)", ansi.Color("testExitReason1", "red+b"), ansi.Color("1", "red+b")),
				fmt.Sprintf("        Last Exit Message: %s", ansi.Color("ContainerError1", "red+b")),
				fmt.Sprintf("        Last Execution Log: \n%s", ansi.Color("TestLog1", "red")),
				"    InitContainer Problems: ",
				fmt.Sprintf("      - Container: %s", ansi.Color("testContainer2", "white+b")),
				fmt.Sprintf("        Status: %s (reason: %s)", ansi.Color("Terminated", "red+b"), ansi.Color("testReason2", "red+b")),
				fmt.Sprintf("        Terminated: %s ago", ansi.Color("0s", "white+b")),
			},
		},
		{
			name: "Critical status pod",
			podProblem: podProblem{
				Name:   "testPod2",
				Status: "Error",
			},
			expectedString: []string{
				fmt.Sprintf("Pod %s:", ansi.Color("testPod2", "white+b")),
				fmt.Sprintf("    Status: %s", ansi.Color("Error", "red+b")),
				fmt.Sprintf("    Created: %s ago", ansi.Color("", "white+b")),
			},
		},
		{
			name: "Uncertain status pod",
			podProblem: podProblem{
				Name:   "testPod3",
				Status: "testStatus",
			},
			expectedString: []string{
				fmt.Sprintf("Pod %s:", ansi.Color("testPod3", "white+b")),
				fmt.Sprintf("    Status: %s", ansi.Color("testStatus", "yellow+b")),
				fmt.Sprintf("    Created: %s ago", ansi.Color("", "white+b")),
			},
		},
	}

	for _, testCase := range testCases {
		result := printPodProblem(&testCase.podProblem)
		assert.Equal(t, result, paddingLeft+strings.Join(testCase.expectedString, paddingLeft+"\n")+"\n", "Unexpected result in testCase %s", testCase.name)
	}
}
