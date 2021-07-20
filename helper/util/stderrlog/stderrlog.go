package stderrlog

import (
	"fmt"
	"io"
	"os"
)

var Writer io.Writer = os.Stderr

func Log(message string) {
	_, _ = fmt.Fprintln(Writer, message)
}

func Logf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(Writer, format+"\n", args...)
}
