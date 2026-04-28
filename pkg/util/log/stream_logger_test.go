package log

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestStreamLoggerStartWaitFallsBackToInfo(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	logger := NewStreamLoggerWithFormat(stdout, stderr, logrus.InfoLevel, RawFormat)

	logger.StartWait("Downloading helm...")
	logger.StopWait()

	expected := "Downloading helm...\n"
	if stdout.String() != expected {
		t.Fatalf("expected %q, got %q", expected, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}
