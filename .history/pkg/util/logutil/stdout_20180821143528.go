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
	err, hasErr := entry.Data[logrus.ErrorKey]
	message := "[" + strings.ToUpper(entry.Level.String()) + "] " + entry.Message + "\n"

	if hasErr {
		errCasted := err.(error)
		message = message + errCasted.Error() + "\n"
	}
	output := []byte(message)

	if entry.Level == logrus.InfoLevel {
		ct.Foreground(Green, false)
		ct.ChangeColor(Red, true, White, false)
		os.Stdout.Write(output)
		ct.ResetColor()
	} else {
		os.Stderr.Write(output)
	}
	return nil
}
