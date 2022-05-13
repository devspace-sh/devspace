package log

import (
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/mgutz/ansi"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"runtime"
)

var baseLog = NewStdoutLogger(os.Stdin, stdout, stderr, logrus.InfoLevel)
var defaultLog = baseLog

//var defaultLog Logger = NewStreamLoggerWithFormat(os.Stdin, logrus.InfoLevel, JsonFormat)

// Discard is a logger implementation that just discards every log statement
var Discard = &DiscardLogger{}

// PrintLogo prints the devspace logo
func PrintLogo() {
	logo := `
     %########%      
     %###########%       ____                 _____                      
         %#########%    |  _ \   ___ __   __ / ___/  ____    ____   ____ ___ 
         %#########%    | | | | / _ \\ \ / / \___ \ |  _ \  / _  | / __// _ \
     %#############%    | |_| |(  __/ \ V /  ____) )| |_) )( (_| |( (__(  __/
     %#############%    |____/  \___|  \_/   \____/ |  __/  \__,_| \___\\___|
 %###############%                                  |_|
 %###########%`
	stdout.Write([]byte(ansi.Color("\r\n"+logo+"\r\n\r\n", "cyan+b")))
}

// StartFileLogging logs the output of the global logger to the file default.log
func StartFileLogging() {
	defaultLog.AddSink(GetFileLogger("default"))
	OverrideRuntimeErrorHandler(false)
}

// GetInstance returns the Logger instance
func GetInstance() Logger {
	return defaultLog
}

// GetBaseInstance returns the base stdout logger
func GetBaseInstance() Logger {
	return baseLog
}

func PrintTable(s Logger, header []string, values [][]string) {
	PrintTableWithOptions(s, header, values, nil)
}

// PrintTableWithOptions prints a table with header columns and string values
func PrintTableWithOptions(s Logger, header []string, values [][]string, modify func(table *tablewriter.Table)) {
	reader, writer := io.Pipe()
	defer writer.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)

		sa := scanner.NewScanner(reader)
		for sa.Scan() {
			s.WriteString(logrus.InfoLevel, "  "+sa.Text()+"\n")
		}
	}()

	table := tablewriter.NewWriter(writer)
	table.SetHeader(header)
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		colors := []tablewriter.Colors{}
		for range header {
			colors = append(colors, tablewriter.Color(tablewriter.FgGreenColor))
		}
		table.SetHeaderColor(colors...)
	}

	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.AppendBulk(values)
	if modify != nil {
		modify(table)
	}

	// Render
	_, _ = writer.Write([]byte("\n"))
	table.Render()
	_, _ = writer.Write([]byte("\n"))
	_ = writer.Close()
	<-done
}
