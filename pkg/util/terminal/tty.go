package terminal

import (
	"os"

	dockerterm "github.com/docker/docker/pkg/term"
	"k8s.io/kubernetes/pkg/kubectl/util/term"
)

// SetupTTY creates a term.TTY (docker)
func SetupTTY() term.TTY {
	t := term.TTY{
		Out: os.Stdout,
		In:  os.Stdin,
	}

	if !t.IsTerminalIn() {
		return t
	}

	// if we get to here, the user wants to attach stdin, wants a TTY, and In is a terminal, so we
	// can safely set t.Raw to true
	t.Raw = true

	stdin, stdout, _ := dockerterm.StdStreams()

	t.In = stdin
	t.Out = stdout

	return t
}
