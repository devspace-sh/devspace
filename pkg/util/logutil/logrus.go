package logutil

import (
	"os"

	"github.com/Sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"
)

var runtimeErrorHandlersOverriden = false

var logs = map[string]*logrus.Logger{}
var terminalHook = TerminalHook{
	LogLevels: []logrus.Level{
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	},
}

func GetLogger(name string, logToTerminal bool) *logrus.Logger {
	log, _ := logs[name]

	if log == nil {
		log = logrus.New()
		log.Formatter = &logrus.JSONFormatter{}

		logdir := "./.devspace/logs/"

		os.MkdirAll(logdir, os.ModePerm)

		logFile, err := os.OpenFile(logdir+name+".log", os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)

		if err != nil {
			log.Warn("Unable to open " + name + " log file. Will log to stdout.")
		} else {
			log.SetOutput(logFile)
		}

		if logToTerminal {
			log.AddHook(terminalHook)
		}
		logs[name] = log
	}

	// We also override the standard runtime error handler
	if runtimeErrorHandlersOverriden == false {
		runtimeErrorHandlersOverriden = true
		errorLog := GetLogger("errors", false)

		if len(runtime.ErrorHandlers) == 2 {
			runtime.ErrorHandlers[0] = func(err error) {
				errorLog.Errorf("Runtime error occurred: %s", err)
			}
		}
	}

	return log
}
