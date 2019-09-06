package analyze

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
)

// ReportItem is the struct that holds the problems
type ReportItem struct {
	Name     string
	Problems []string
}

// HeaderWidth is the width of the header
const HeaderWidth = 80

// PaddingLeft is the left padding of the report
const PaddingLeft = 2

// HeaderChar is the header char used
const HeaderChar = "="

var paddingLeft = newString(" ", PaddingLeft)

// Analyze analyses a given
func Analyze(client *kubectl.Client, namespace string, noWait bool, log log.Logger) error {
	report, err := CreateReport(client, namespace, noWait)
	if err != nil {
		return err
	}

	reportString := ReportToString(report)
	log.WriteString(reportString)

	return nil
}

// CreateReport creates a new report about a certain namespace
func CreateReport(client *kubectl.Client, namespace string, noWait bool) ([]*ReportItem, error) {
	report := []*ReportItem{}

	// Analyze pods
	problems, err := Pods(client, namespace, noWait)
	if err != nil {
		return nil, fmt.Errorf("Error during analyzing pods: %v", err)
	}
	if len(problems) > 0 {
		report = append(report, &ReportItem{
			Name:     "Pods",
			Problems: problems,
		})
	}

	// We only check events if we suspect a problem
	checkEvents := len(report) > 0

	// Analyze replicasets
	if checkEvents == false {
		replicaSetProblems, err := ReplicaSets(client.Client, namespace)
		if err != nil {
			return nil, fmt.Errorf("Error during analyzing replica sets: %v", err)
		}
		if len(replicaSetProblems) > 0 {
			checkEvents = true
		}
	}

	// Analyze statefulsets
	if checkEvents == false {
		statefulSetProblems, err := StatefulSets(client.Client, namespace)
		if err != nil {
			return nil, fmt.Errorf("Error during analyzing stateful sets: %v", err)
		}
		if len(statefulSetProblems) > 0 {
			checkEvents = true
		}
	}

	if checkEvents {
		// Analyze events
		problems, err = Events(client.Client, namespace)
		if err != nil {
			return nil, fmt.Errorf("Error during analyzing events: %v", err)
		}
		if len(problems) > 0 {
			// Prepend to report
			report = append([]*ReportItem{&ReportItem{
				Name:     "Events",
				Problems: problems,
			}}, report...)
		}
	}

	return report, nil
}

// ReportToString transforms a report to a string
func ReportToString(report []*ReportItem) string {
	reportString := ""

	if len(report) == 0 {
		reportString += fmt.Sprintf("\n%sNo problems found.\n%sRun `%s` if you want show pod logs\n\n", paddingLeft, paddingLeft, ansi.Color("devspace logs -p", "white+b"))
	} else {
		reportString += "\n"

		for _, reportItem := range report {
			reportString += createHeader(reportItem.Name, len(reportItem.Problems))

			for _, problem := range reportItem.Problems {
				reportString += problem + "\n"
			}
		}
	}

	return reportString
}

func createHeader(name string, problemCount int) string {
	header := fmt.Sprintf(" %s (%d potential issue(s)) ", name, problemCount)
	if len(header)%2 == 1 {
		header += " "
	}

	padding := HeaderWidth - len(header)
	header = newString(" ", padding/2) + header + newString(" ", padding/2)

	return ansi.Color(fmt.Sprintf("%s\n%s\n%s\n", paddingLeft+newString(HeaderChar, HeaderWidth), paddingLeft+header, paddingLeft+newString(HeaderChar, HeaderWidth)), "green+b")
}

func newString(char string, size int) string {
	retStr := ""
	for i := 0; i < size; i++ {
		retStr += char
	}

	return retStr
}
