package log

import (
	"strings"

	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
)

var defaultLog Logger = &stdoutLogger{
	level: logrus.DebugLevel,
}

// Discard is a logger implementation that just discards every log statement
var Discard = &DiscardLogger{}

// StartWait prints a wait message until StopWait is called
func StartWait(message string) {
	defaultLog.StartWait(message)
}

// StopWait stops printing the wait message
func StopWait() {
	defaultLog.StopWait()
}

// PrintLogo prints the devspace logo
func PrintLogo() {
	logo := `
     ____              ____                       
    |  _ \  _____   __/ ___| _ __   __ _  ___ ___ 
    | | | |/ _ \ \ / /\___ \| '_ \ / _` + "`" + ` |/ __/ _ \
    | |_| |  __/\ V /  ___) | |_) | (_| | (_|  __/
    |____/ \___| \_/  |____/| .__/ \__,_|\___\___|
                            |_|`

	stdout.Write([]byte(ansi.Color(logo+"\r\n\r\n", "cyan+b")))
}

// Debug prints debug information
func Debug(args ...interface{}) {
	defaultLog.Debug(args...)
}

// Debugf prints formatted debug information
func Debugf(format string, args ...interface{}) {
	defaultLog.Debugf(format, args...)
}

// Info prints info information
func Info(args ...interface{}) {
	defaultLog.Info(args...)
}

// Infof prints formatted information
func Infof(format string, args ...interface{}) {
	defaultLog.Infof(format, args...)
}

// Warn prints warning information
func Warn(args ...interface{}) {
	defaultLog.Warn(args...)
}

// Warnf prints formatted warning information
func Warnf(format string, args ...interface{}) {
	defaultLog.Warnf(format, args...)
}

// Error prints error information
func Error(args ...interface{}) {
	defaultLog.Error(args...)
}

// Errorf prints formatted error information
func Errorf(format string, args ...interface{}) {
	defaultLog.Errorf(format, args...)
}

// Fatal prints fatal error information
func Fatal(args ...interface{}) {
	defaultLog.Fatal(args...)
}

// Fatalf prints formatted fatal error information
func Fatalf(format string, args ...interface{}) {
	defaultLog.Fatalf(format, args...)
}

// Panic prints panic information
func Panic(args ...interface{}) {
	defaultLog.Panic(args...)
}

// Panicf prints formatted panic information
func Panicf(format string, args ...interface{}) {
	defaultLog.Panicf(format, args...)
}

// Done prints done information
func Done(args ...interface{}) {
	defaultLog.Done(args...)
}

// Donef prints formatted info information
func Donef(format string, args ...interface{}) {
	defaultLog.Donef(format, args...)
}

// Fail prints error information
func Fail(args ...interface{}) {
	defaultLog.Fail(args...)
}

// Failf prints formatted error information
func Failf(format string, args ...interface{}) {
	defaultLog.Failf(format, args...)
}

// Print prints information
func Print(level logrus.Level, args ...interface{}) {
	defaultLog.Print(level, args...)
}

// Printf prints formatted information
func Printf(level logrus.Level, format string, args ...interface{}) {
	defaultLog.Printf(level, format, args...)
}

// SetLevel changes the log level of the global logger
func SetLevel(level logrus.Level) {
	defaultLog.SetLevel(level)
}

// StartFileLogging logs the output of the global logger to the file default.log
func StartFileLogging() {
	defaultLogStdout, ok := defaultLog.(*stdoutLogger)
	if ok {
		defaultLogStdout.fileLogger = GetFileLogger("default")
	}

	OverrideRuntimeErrorHandler(false)
}

// GetInstance returns the Logger instance
func GetInstance() Logger {
	return defaultLog
}

// SetInstance sets the default logger instance
func SetInstance(logger Logger) {
	defaultLog = logger
}

// WriteColored writes a message in color
func WriteColored(message string, color string) {
	defaultLog.Write([]byte(ansi.Color(message, color)))
}

// Write writes to the stdout log without formatting the message, but takes care of locking the log and halting a possible wait message
func Write(message []byte) {
	defaultLog.Write(message)
}

// WriteString writes to the stdout log without formatting the message, but takes care of locking the log and halting a possible wait message
func WriteString(message string) {
	defaultLog.WriteString(message)
}

//SetFakePrintTable is a testing tool that allows overwriting the function PrintTable
func SetFakePrintTable(fake func(s Logger, header []string, values [][]string)) {
	fakePrintTable = fake
}

var fakePrintTable func(s Logger, header []string, values [][]string)

// PrintTable prints a table with header columns and string values
func PrintTable(s Logger, header []string, values [][]string) {
	if fakePrintTable != nil {
		fakePrintTable(s, header, values)
		return
	}

	columnLengths := make([]int, len(header))

	for k, v := range header {
		columnLengths[k] = len(v)
	}

	// Get maximum column length
	for _, v := range values {
		for key, value := range v {
			if len(value) > columnLengths[key] {
				columnLengths[key] = len(value)
			}
		}
	}

	s.Write([]byte("\n"))

	// Print Header
	for key, value := range header {
		WriteColored(" "+value+"  ", "green+b")

		padding := columnLengths[key] - len(value)

		if padding > 0 {
			s.Write([]byte(strings.Repeat(" ", padding)))
		}
	}

	s.Write([]byte("\n"))

	if len(values) == 0 {
		s.Write([]byte(" No entries found\n"))
	}

	// Print Values
	for _, v := range values {
		for key, value := range v {
			s.Write([]byte(" " + value + "  "))

			padding := columnLengths[key] - len(value)

			if padding > 0 {
				s.Write([]byte(strings.Repeat(" ", padding)))
			}
		}

		s.Write([]byte("\n"))
	}

	s.Write([]byte("\n"))
}
