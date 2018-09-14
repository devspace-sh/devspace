package stdinutil

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/daviddengcn/go-colortext"

	"github.com/covexo/devspace/pkg/util/paramutil"
)

//GetFromStdinParams defines a question and its answerpatterns
type GetFromStdinParams struct {
	Question               string
	DefaultValue           string
	ValidationRegexPattern string
	InputTerminationString string
}

var defaultParams = &GetFromStdinParams{
	ValidationRegexPattern: ".*",
	InputTerminationString: "\n",
}

var reader *bufio.Reader

const changeQuestion = "Would you like to change it? (yes, no/ENTER))"

//GetFromStdin asks the user a question and returns the answer
func GetFromStdin(params *GetFromStdinParams) *string {
	paramutil.SetDefaults(params, defaultParams)

	validationRegexp, _ := regexp.Compile(params.ValidationRegexPattern)

	if reader == nil {
		reader = bufio.NewReader(os.Stdin)

		defer func() {
			reader = nil
		}()
	}
	input := ""

	for {
		fmt.Print(params.Question)

		if len(params.DefaultValue) > 0 {
			fmt.Print("\n")
			log.WriteColored("Press ENTER to use: "+params.DefaultValue, ct.Green)
		}
		fmt.Print("\n")

		for {
			fmt.Print("> ")
			nextLine, _ := reader.ReadString('\n')
			nextLine = strings.Trim(nextLine, "\r\n ")

			if strings.Compare(params.InputTerminationString, "\n") == 0 {
				// Assign the input value to input var
				input = nextLine
				break
			}
			input += nextLine + "\n"

			if strings.HasSuffix(input, params.InputTerminationString+"\n") {
				input = strings.TrimSuffix(input, params.InputTerminationString+"\n")
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
			input = ""
		}
	}
	fmt.Println("")

	return &input
}

//AskChangeQuestion asks two questions. Do you want to change this value? If yes, what's the new value?
func AskChangeQuestion(params *GetFromStdinParams) *string {
	paramutil.SetDefaults(params, defaultParams)

	if reader == nil {
		reader = bufio.NewReader(os.Stdin)

		defer func() {
			reader = nil
		}()
	}

	shouldValueChangeQuestion := GetFromStdinParams{
		Question:               params.Question + "\nThis is the current value:\n#################\n" + strings.TrimRight(params.DefaultValue, "\r\n") + "\n#################\n" + changeQuestion,
		DefaultValue:           "no",
		ValidationRegexPattern: "yes|no",
	}

	shouldChangeAnswer := GetFromStdin(&shouldValueChangeQuestion)

	if *shouldChangeAnswer == "no" {
		return &params.DefaultValue
	}

	newValueQuestion := GetFromStdinParams{
		Question:               "Please enter the new value:",
		DefaultValue:           params.DefaultValue,
		ValidationRegexPattern: params.ValidationRegexPattern,
		InputTerminationString: params.InputTerminationString,
	}

	newValue := GetFromStdin(&newValueQuestion)
	return newValue
}
