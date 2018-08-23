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

var reader *bufio.Reader

const changeQuestion = "Would you like to change it? (yes, no/ENTER))"

func GetFromStdin(params *GetFromStdin_params) string {
	paramutil.SetDefaults(params, defaultParams)

	validationRegexp, _ := regexp.Compile(params.ValidationRegexPattern)
	if reader == nil {
		reader = bufio.NewReader(os.Stdin)
	}
	defer func() {
		reader = nil
	}()
	input := ""

	for {
		fmt.Print(params.Question)

		if len(params.DefaultValue) > 0 {
			fmt.Print("\n")
			fmt.Print("Press ENTER to use: " + params.DefaultValue + "")
		}
		fmt.Print("\n")

		for {
			fmt.Print("> ")
			nextLine, _ := reader.ReadString('\n')
			nextLine = strings.Trim(nextLine, "\r\n ")

			fmt.Println("Input: " + nextLine)

			if strings.Compare(params.InputTerminationString, "\n") == 0 {
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

func AskChangeQuestion(params *GetFromStdin_params) string{
	paramutil.SetDefaults(params, defaultParams)

	shouldValueChangeQuestion := GetFromStdin_params{
		Question: params.Question + "\nThis is the current value:\n#################\n" + strings.TrimRight(params.DefaultValue, "\r\n") + "\n#################\n" + changeQuestion,
		DefaultValue: "no",
		ValidationRegexPattern: "yes|no",
	}

	shouldChangeAnswer := GetFromStdin(&shouldValueChangeQuestion)

	if shouldChangeAnswer == "no" {
		return params.DefaultValue
	}

	newValueQuestion := GetFromStdin_params{
		Question: "Please enter the new value:",
		DefaultValue: params.DefaultValue,
		ValidationRegexPattern: params.ValidationRegexPattern,
		InputTerminationString: params.InputTerminationString,
	}

	return GetFromStdin(&newValueQuestion)
}
