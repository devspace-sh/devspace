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
		ct.Foreground(ct.Green, false)
		ct.ChangeColor(ct.Red, true, ct.White, false)
		errCasted := err.(error)
		message = message + errCasted.Error() + "\n"
		ct.ResetColor()
	}
	output := []byte(message)

	if entry.Level == logrus.InfoLevel {
		ct.Foreground(ct.Green, false)
		ct.ChangeColor(ct.Red, true, ct.White, false)
		os.Stdout.Write(output)
		ct.ResetColor()
	} else {
		os.Stderr.Write(output)
	}
	return nil
}
