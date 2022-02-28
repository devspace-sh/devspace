package log

import (
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

var defaultLog Logger = NewStdoutLogger(os.Stdin, stdout, logrus.InfoLevel)

//var defaultLog Logger = NewStreamLoggerWithFormat(os.Stdin, logrus.InfoLevel, JsonFormat)

// Discard is a logger implementation that just discards every log statement
var Discard = &DiscardLogger{}

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

// StartFileLogging logs the output of the global logger to the file default.log
func StartFileLogging() {
	defaultLog = NewUnionLogger(defaultLog, GetFileLogger("default"))
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
func writeColored(message string, color string) {
	defaultLog.WriteString(logrus.InfoLevel, ansi.Color(message, color))
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
			if len(value) > 64 {
				value = value[:61] + "..."
				v[key] = value
			}

			if len(value) > columnLengths[key] {
				columnLengths[key] = len(value)
			}
		}
	}

	s.WriteString(logrus.InfoLevel, "\n")

	// Print Header
	for key, value := range header {
		writeColored(" "+value+"  ", "green+b")

		padding := columnLengths[key] - len(value)

		if padding > 0 {
			s.WriteString(logrus.InfoLevel, strings.Repeat(" ", padding))
		}
	}

	s.WriteString(logrus.InfoLevel, "\n")

	if len(values) == 0 {
		s.WriteString(logrus.InfoLevel, " No entries found\n")
	}

	// Print Values
	for _, v := range values {
		for key, value := range v {
			s.WriteString(logrus.InfoLevel, " "+value+"  ")

			padding := columnLengths[key] - len(value)

			if padding > 0 {
				s.WriteString(logrus.InfoLevel, strings.Repeat(" ", padding))
			}
		}

		s.WriteString(logrus.InfoLevel, "\n")
	}

	s.WriteString(logrus.InfoLevel, "\n")
}
