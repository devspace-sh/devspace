package logutil

import (
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	ct "github.com/daviddengcn/go-colortext"
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
		ct.Foreground(ct.Red, false)
		errCasted := err.(error)
		message = message + errCasted.Error() + "\n"
		ct.ResetColor()
	}
	output := []byte(message)

	if entry.Level == logrus.InfoLevel {
		ct.Foreground(ct.Green, false)
		os.Stdout.Write(output)
		ct.ResetColor()
	} else {
		ct.Foreground(ct.Red, false)
		os.Stderr.Write(output)
		ct.ResetColor()
	}
	return nil
}
