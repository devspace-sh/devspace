package analyze

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
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
	Analyze(namespace string, options Options) error
	CreateReport(namespace string, options Options) ([]*ReportItem, error)
}

// Options is the options to pass to the analyzer
type Options struct {
	Wait    bool
	Timeout int
	Patient bool

	IgnorePodRestarts bool
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
func (a *analyzer) Analyze(namespace string, options Options) error {
	report, err := a.CreateReport(namespace, options)
	if err != nil {
		return err
	}

	reportString := ReportToString(report)
	a.log.WriteString(logrus.InfoLevel, reportString)

	return nil
}

// CreateReport creates a new report about a certain namespace
func (a *analyzer) CreateReport(namespace string, options Options) ([]*ReportItem, error) {
	a.log.Info("Checking status...")

	report := []*ReportItem{}
	timeout := WaitTimeout
	if options.Timeout > 0 {
		timeout = time.Duration(options.Timeout) * time.Second
	}

	// Loop as long as we have a timeout
	err := wait.Poll(time.Second, timeout, func() (bool, error) {
		report = []*ReportItem{}

		// Analyze pods
		problems, err := a.pods(namespace, options)
		if err != nil {
			return false, errors.Errorf("Error during analyzing pods: %v", err)
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
		if !checkEvents {
			replicaSetProblems, err := a.replicaSets(namespace)
			if err != nil {
				return false, errors.Errorf("Error during analyzing replica sets: %v", err)
			}
			if len(replicaSetProblems) > 0 {
				checkEvents = true
			}
		}

		// Analyze statefulsets
		if !checkEvents {
			statefulSetProblems, err := a.statefulSets(namespace)
			if err != nil {
				return false, errors.Errorf("Error during analyzing stateful sets: %v", err)
			}
			if len(statefulSetProblems) > 0 {
				checkEvents = true
			}
		}

		if checkEvents {
			// Analyze events
			problems, err = a.events(namespace)
			if err != nil {
				return false, errors.Errorf("Error during analyzing events: %v", err)
			}
			if len(problems) > 0 {
				// Prepend to report
				report = append([]*ReportItem{{
					Name:     "Events",
					Problems: problems,
				}}, report...)
			}
		}

		return len(report) == 0 || !options.Wait || !options.Patient, nil
	})
	if err != nil && len(report) == 0 {
		return nil, err
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
