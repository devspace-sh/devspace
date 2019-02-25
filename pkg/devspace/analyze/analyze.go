package analyze

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type reportItem struct {
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
func Analyze(client *kubernetes.Clientset, config *rest.Config, namespace string, noWait bool, log log.Logger) error {
	report := []*reportItem{}

	// Analyze events
	problems, err := Events(client, config, namespace)
	if err != nil {
		return fmt.Errorf("Error during analyzing events: %v", err)
	}
	if len(problems) > 0 {
		report = append(report, &reportItem{
			Name:     "Events",
			Problems: problems,
		})
	}

	// Analyze pods
	problems, err = Pods(client, namespace, noWait)
	if err != nil {
		return fmt.Errorf("Error during analyzing pods: %v", err)
	}
	if len(problems) > 0 {
		report = append(report, &reportItem{
			Name:     "Pods",
			Problems: problems,
		})
	}

	printReport(report, log)
	return nil
}

func printReport(report []*reportItem, log log.Logger) {
	if len(report) == 0 {
		log.WriteString(fmt.Sprintf("\n%sNo problems found.\n%sRun `%s` if you want show pod logs\n\n", paddingLeft, paddingLeft, ansi.Color("devspace logs -p", "white+b")))
	} else {
		log.WriteString("\n")

		for _, reportItem := range report {
			log.WriteString(createHeader(reportItem.Name, len(reportItem.Problems)))

			for _, problem := range reportItem.Problems {
				log.WriteString(problem + "\n")
			}
		}
	}
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
