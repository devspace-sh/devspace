package analyze

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
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

// Analyzer is the interface for analyzing
type Analyzer interface {
	Analyze(namespace string, noWait bool) error
	CreateReport(namespace string, noWait bool) ([]*ReportItem, error)
}

type analyzer struct {
	client kubectl.Client
	log    log.Logger
}

// NewAnalyzer creates a new analyzer from the kube client
func NewAnalyzer(client kubectl.Client, log log.Logger) Analyzer {
	return &analyzer{
		client: client,
		log:    log,
	}
}

// Analyze analyses a given
func (a *analyzer) Analyze(namespace string, noWait bool) error {
	report, err := a.CreateReport(namespace, noWait)
	if err != nil {
		return err
	}

	reportString := ReportToString(report)
	a.log.WriteString(reportString)

	return nil
}

// CreateReport creates a new report about a certain namespace
func (a *analyzer) CreateReport(namespace string, noWait bool) ([]*ReportItem, error) {
	report := []*ReportItem{}

	// Analyze pods
	problems, err := a.pods(namespace, noWait)
	if err != nil {
		return nil, errors.Errorf("Error during analyzing pods: %v", err)
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
		replicaSetProblems, err := a.replicaSets(namespace)
		if err != nil {
			return nil, errors.Errorf("Error during analyzing replica sets: %v", err)
		}
		if len(replicaSetProblems) > 0 {
			checkEvents = true
		}
	}

	// Analyze statefulsets
	if checkEvents == false {
		statefulSetProblems, err := a.statefulSets(namespace)
		if err != nil {
			return nil, errors.Errorf("Error during analyzing stateful sets: %v", err)
		}
		if len(statefulSetProblems) > 0 {
			checkEvents = true
		}
	}

	if checkEvents {
		// Analyze events
		problems, err = a.events(namespace)
		if err != nil {
			return nil, errors.Errorf("Error during analyzing events: %v", err)
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
		reportString += fmt.Sprintf("\n%sNo problems found.\n%sRun `%s` if you want show pod logs\n\n", paddingLeft, paddingLeft, ansi.Color("devspace logs --pick", "white+b"))
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
