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

	level := "[" + strings.ToUpper(entry.Level.String()) + "]   "
	message := entry.Message + "\n"

	if entry.Level == logrus.DebugLevel {
		ct.Foreground(ct.Green, false)
	} else if entry.Level == logrus.InfoLevel {
		ct.Foreground(ct.Green, false)
	} else if entry.Level == logrus.WarnLevel {
		ct.Foreground(ct.Red, false)
	} else {
		ct.Foreground(ct.Red, false)
	}

	if hasErr {
		errCasted := err.(error)
		message = message + errCasted.Error() + "\n"
	}

	output := []byte(message)

	if entry.Level == logrus.InfoLevel {
		os.Stdout.Write([]byte(level))
		ct.ResetColor()
		os.Stdout.Write(output)
	} else {
		os.Stderr.Write([]byte(level))
		ct.ResetColor()
		os.Stderr.Write(output)
	}

	return nil
}
