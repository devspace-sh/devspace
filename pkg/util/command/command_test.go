package command

import (
	"context"
	"testing"
)

func TestCommand(t *testing.T) {
	fakeCommand := &FakeCommand{}
	err := fakeCommand.Run("", nil, nil, nil)
	if err != nil {
		t.Fatalf("FakeCommand unexpectedly returned error: %v", err)
	}

	streamCommand := newStreamCommand("echo", []string{"hello"})
	err = streamCommand.Run(context.Background(), "", nil, nil, nil)
	if err != nil {
		t.Fatalf("StreamCommand unexpectedly returned error: %v", err)
	}
}
