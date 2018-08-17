package logutil

import (
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
)

type TerminalHook struct {
	LogLevels []logrus.Level
}

func (hook TerminalHook) Levels() []logrus.Level {
	return hook.LogLevels
}

func (hook TerminalHook) Fire(entry *logrus.Entry) error {
	message := "[" + strings.ToUpper(entry.Level.String()) + "] " + entry.Message + "\n"
	output := []byte(message)

	if entry.Level == logrus.InfoLevel {
		os.Stdout.Write(output)
	} else {
		os.Stderr.Write(output)
	}
	return nil
}
