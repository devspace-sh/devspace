package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/devspace-cloud/devspace/e2e/tests/analyze"
	"github.com/devspace-cloud/devspace/e2e/tests/build"
	"github.com/devspace-cloud/devspace/e2e/tests/deploy"
	"github.com/devspace-cloud/devspace/e2e/tests/dev"
	"github.com/devspace-cloud/devspace/e2e/tests/enter"
	"github.com/devspace-cloud/devspace/e2e/tests/examples"
	"github.com/devspace-cloud/devspace/e2e/tests/initcmd"
	"github.com/devspace-cloud/devspace/e2e/tests/logs"
	"github.com/devspace-cloud/devspace/e2e/tests/print"
	"github.com/devspace-cloud/devspace/e2e/tests/render"
	"github.com/devspace-cloud/devspace/e2e/tests/run"
	"github.com/devspace-cloud/devspace/e2e/tests/space"
	"github.com/devspace-cloud/devspace/e2e/tests/sync"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

var testNamespace = "testing-test-namespace"

// Create a new type for a list of Strings
type stringList []string

// Implement the flag.Value interface
func (s *stringList) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringList) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

type Test interface {
	Run(subTests []string, ns string, pwd string, logger log.Logger, verbose bool, timeout int) error
	SubTests() []string
}

var availableTests = map[string]Test{
	"analyze":  analyze.RunNew,
	"build":    build.RunNew,
	"deploy":   deploy.RunNew,
	"dev":      dev.RunNew,
	"enter":    enter.RunNew,
	"examples": examples.RunNew,
	"init":     initcmd.RunNew,
	"logs":     logs.RunNew,
	"print":    print.RunNew,
	"render":   render.RunNew,
	"run":      run.RunNew,
	"space":    space.RunNew,
	"sync":     sync.RunNew,
}

var subTests = map[string]*stringList{}

func main() {
	logger := log.GetInstance()
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	testCommand := flag.NewFlagSet("test", flag.ExitOnError)
	purgeNamespacesCommand := flag.NewFlagSet("purge-namespaces", flag.ExitOnError)
	listCommand := flag.NewFlagSet("list", flag.ExitOnError)

	for t := range availableTests {
		subTests[t] = &stringList{}
		testCommand.Var(subTests[t], "test-"+t, "A comma seperated list of sub tests to be passed")
	}

	var test stringList
	testCommand.Var(&test, "test", "A comma seperated list of group tests to pass")

	var skiptest stringList
	testCommand.Var(&skiptest, "skip-test", "A comma seperated list of group tests to skip")

	var verbose bool
	testCommand.BoolVar(&verbose, "verbose", false, "Displays the tests outputs in real time (default: false)")

	var timeout int
	testCommand.IntVar(&timeout, "timeout", 200, "Sets a timeout limit in seconds for each test (default: 200)")

	var testlist bool
	testCommand.BoolVar(&testlist, "list", false, "Displays a list of sub commands")

	// Verify that a subcommand has been provided
	// os.Arg[0] is the main command
	// os.Arg[1] will be the subcommand
	if len(os.Args) < 2 {
		fmt.Println("test or list subcommand is required")
		os.Exit(1)
	}

	// Switch on the subcommand
	// Parse the flags for appropriate FlagSet
	// FlagSet.Parse() requires a set of arguments to parse as input
	// os.Args[2:] will be all arguments starting after the subcommand at os.Args[1]
	switch os.Args[1] {
	case "list":
		listCommand.Parse(os.Args[2:])
	case "test":
		testCommand.Parse(os.Args[2:])
	case "purge-namespaces":
		purgeNamespacesCommand.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	// FlagSet.Parse() will evaluate to false if no flags were parsed (i.e. the user did not provide any flags)
	// If "list" and "test" are used together, only the former will be parsed and recognized, the latter will be ignored
	if listCommand.Parsed() {
		// Required Flags
		fmt.Println("List of available commands:")
		fmt.Println("\t - test: \t\tRuns all the tests sequentially (use --list to display a list of sub commands)")
		fmt.Println("\t - purge-namespaces: \tDeletes namespaces that might have failed to be deleted during previous test runs")
	}
	if testCommand.Parsed() {

		// Handles 'list' command
		if testlist {
			// skip-test, verbose, timeout, test
			fmt.Println("List of available sub commands for the 'test' command:")
			// --test
			fmt.Printf("\t --test: A comma seperated list of group tests to pass [ ")
			for key := range availableTests {
				fmt.Printf("%v ", key)
			}
			fmt.Printf("]\n ")

			// --skip-test
			fmt.Printf("\t --skip-test: A comma seperated list of group tests to skip [ ")
			for key := range availableTests {
				fmt.Printf("%v ", key)
			}
			fmt.Printf("]\n ")

			// --test-xxx
			for testName, testRun := range availableTests {
				fmt.Printf("\t --test-%s: A comma seperated list of sub tests to pass for the '%s' group test [ ", testName, testName)
				for _, st := range testRun.SubTests() {
					fmt.Printf("%v ", st)
				}
				fmt.Printf("]\n ")
			}

			// --verbose
			fmt.Println("\n\t --verbose: Displays tests output in real time (default: false)")
			// --timeout
			fmt.Println("\t --timeout: Sets a timeout limit in seconds for each test (default: 200)")

			return
		}

		if len(test) > 0 && len(skiptest) > 0 {
			logger.Error("flags '--test' and '--skip-test' cannot be used together")
			os.Exit(1)
		}

		var testsToRun = map[string]Test{}
		// We gather all the group tests called with the --test flag. e.g: --test=examples,init
		for _, testName := range test {
			if availableTests[testName] == nil {
				// arg is not valid
				fmt.Printf("'%v' is not a valid argument for --test. Valid arguments are the following: [ ", testName)
				for key := range availableTests {
					fmt.Printf("%v ", key)
				}
				fmt.Printf("]\n ")
				os.Exit(1)
			}
			testsToRun[testName] = availableTests[testName]
		}

		// If cmd test with --test-xxx
		if len(testsToRun) == 0 {
			for testName, args := range subTests {
				if args != nil && len(*args) > 0 {
					test = append(test, testName)
					testsToRun[testName] = availableTests[testName]
				}
			}
			// Sorts tests alphabetically
			sort.Strings(test)
		}

		// If cmd test alone (if no --test flag), we want to run all available tests
		if len(testsToRun) == 0 {
			for testName := range availableTests {
				test = append(test, testName)
				testsToRun[testName] = availableTests[testName]
			}
			// Sorts tests alphabetically
			sort.Strings(test)
		}

		// --skip-test
		for _, testName := range skiptest {
			if availableTests[testName] == nil {
				// arg is not valid
				fmt.Printf("'%v' is not a valid argument for --skip-test. Valid arguments are the following: [ ", testName)
				for key := range availableTests {
					fmt.Printf("%v ", key)
				}
				fmt.Printf("]\n ")
				os.Exit(1)
			}
			delete(testsToRun, testName)
		}

		for _, tName := range test {
			// for testName, testRun := range testsToRun {
			parameterSubTests := []string{}
			// --test-xxx sub command
			if t, ok := subTests[tName]; ok && t != nil && len(*t) > 0 {
				for _, s := range *t {
					if !utils.StringInSlice(s, testsToRun[tName].SubTests()) {
						// arg is not valid
						fmt.Printf("'%v' is not a valid argument for --test-%v. Valid arguments are the following: [ ", s, tName)
						for _, st := range testsToRun[tName].SubTests() {
							fmt.Printf("%v ", st)
						}
						fmt.Printf("]\n ")
						os.Exit(1)
					}
					// deploy,init,sync,logs,examples,space
					parameterSubTests = append(parameterSubTests, s)
				}
			}

			// We run the actual group tests by passing the sub tests
			err := testsToRun[tName].Run(parameterSubTests, testNamespace, pwd, logger, verbose, timeout)
			if err != nil {
				logger.Error(err)
				os.Exit(1)
			}
		}
	}
	if purgeNamespacesCommand.Parsed() {
		var nsPrefixes []string

		for t := range availableTests {
			nsPrefixes = append(nsPrefixes, "test-"+t)
		}

		err := utils.PurgeNamespacesByPrefixes(nsPrefixes)
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	}
}
