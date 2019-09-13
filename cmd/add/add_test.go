package add

import (
	"fmt"
	"strings"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"gotest.tools/assert"
)

var logOutput string

type testLogger struct {
	log.DiscardLogger
}

func (t testLogger) Info(args ...interface{}) {
	logOutput = logOutput + "\nInfo " + fmt.Sprint(args...)
}
func (t testLogger) Infof(format string, args ...interface{}) {
	logOutput = logOutput + "\nInfo " + fmt.Sprintf(format, args...)
}

func (t testLogger) Done(args ...interface{}) {
	logOutput = logOutput + "\nDone " + fmt.Sprint(args...)
}
func (t testLogger) Donef(format string, args ...interface{}) {
	logOutput = logOutput + "\nDone " + fmt.Sprintf(format, args...)
}

func (t testLogger) Warn(args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprint(args...)
}
func (t testLogger) Warnf(format string, args ...interface{}) {
	logOutput = logOutput + "\nWarn " + fmt.Sprintf(format, args...)
}

func TestAdd(t *testing.T) {
	addCmd := NewAddCmd(&flags.GlobalFlags{})
	subcommands := addCmd.Commands()

	expectedSubcommandNames := []string{"deployment", "image", "port", "provider", "sync"}
	for _, subcommand := range subcommands {
		subCommandName := subcommand.Name()
		index := pos(expectedSubcommandNames, subCommandName)
		assert.Equal(t, true, index > -1, "Wrong subcommand "+subCommandName)
		expectedSubcommandNames = append(expectedSubcommandNames[:index], expectedSubcommandNames[index+1:]...)
	}
	assert.Equal(t, 0, len(expectedSubcommandNames), "Some subcommands of add are missing: "+strings.Join(expectedSubcommandNames, ", "))
}

func pos(slice []string, value string) int {
	for p, v := range slice {
		if v == value {
			return p
		}
	}
	return -1
}
