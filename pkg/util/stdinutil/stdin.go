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
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/pkg/term"
)

//GetFromStdinParams defines a question and its answerpatterns
type GetFromStdinParams struct {
	Question               string
	DefaultValue           string
	ValidationRegexPattern string
	InputTerminationString string
	IsPassword             bool
}

var defaultParams = &GetFromStdinParams{
	ValidationRegexPattern: ".*",
	InputTerminationString: "\n",
}

const changeQuestion = "Would you like to change it? (yes, no/ENTER))"

//GetFromStdin asks the user a question and returns the answer
func GetFromStdin(params *GetFromStdinParams) *string {
	paramutil.SetDefaults(params, defaultParams)

	validationRegexp, _ := regexp.Compile(params.ValidationRegexPattern)
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

			reader := bufio.NewReader(os.Stdin)
			nextLine := ""

			if params.IsPassword {
				inStreamFD := command.NewInStream(os.Stdin).FD()
				oldState, err := term.SaveState(inStreamFD)
				if err != nil {
					log.Fatal(err)
				}

				term.DisableEcho(inStreamFD, oldState)
				nextLine, _ = reader.ReadString('\n')
				term.RestoreTerminal(inStreamFD, oldState)
			} else {
				nextLine, _ = reader.ReadString('\n')
			}

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
