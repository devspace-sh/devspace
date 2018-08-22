package stdinutil

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/covexo/devspace/pkg/util/paramutil"
)

type GetFromStdin_params struct {
	Question               string
	DefaultValue           string
	ValidationRegexPattern string
	InputTerminationString string
}

var defaultParams = &GetFromStdin_params{
	ValidationRegexPattern: ".*",
	InputTerminationString: "\n",
}

func GetFromStdin(params *GetFromStdin_params) string {
	paramutil.SetDefaults(params, defaultParams)

	validationRegexp, _ := regexp.Compile(params.ValidationRegexPattern)
	reader := bufio.NewReader(os.Stdin)
	input := ""

	for {
		fmt.Print(params.Question)

		changeQuestion := "Would you like to change it? (yes, no/ENTER))"
		isChangeQuestion := false

		if len(params.DefaultValue) > 0 {
			fmt.Print("\n")

			if params.InputTerminationString == "\n" {
				fmt.Print("Press ENTER to use: " + params.DefaultValue + "")
			} else {
				fmt.Println("This is the current value:\n#################\n" + strings.TrimRight(params.DefaultValue, "\r\n") + "\n#################")
				fmt.Print(changeQuestion)
				isChangeQuestion = true
			}
		}
		fmt.Print("\n")

		for {
			fmt.Print("> ")
			nextLine, _ := reader.ReadString('\n')
			nextLine = strings.Trim(nextLine, "\r\n ")

			if isChangeQuestion {
				if nextLine == "yes" {
					isChangeQuestion = false
					fmt.Println("Please enter the new value:")
				} else if len(nextLine) == 0 || nextLine == "no" {
					break
				} else {
					fmt.Println(changeQuestion)
				}
				continue
			}

			if params.InputTerminationString == "\n" {
				input = nextLine
				break
			}
			input += nextLine + "\n"

			if strings.HasSuffix(nextLine, params.InputTerminationString) {
				break
			}
		}

		if len(input) == 0 && len(params.DefaultValue) > 0 {
			input = params.DefaultValue
		}

		if validationRegexp.MatchString(input) {
			break
		} else {
			fmt.Print("Input must match " + params.ValidationRegexPattern + "\n")
		}
	}
	return input
}
