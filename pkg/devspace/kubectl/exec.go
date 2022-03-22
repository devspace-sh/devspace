package kubectl

import (
	"bytes"
	"context"
	"github.com/loft-sh/devspace/pkg/util/terminal"
	"io"
	"k8s.io/kubectl/pkg/util/term"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	kubectlExec "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/scheme"
)

// SubResource specifies with sub resources should be used for the container connection (exec or attach)
type SubResource string

const (
	// SubResourceExec creates a new process in the container and attaches to that
	SubResourceExec SubResource = "exec"

	// SubResourceAttach attaches to the top process of the container
	SubResourceAttach SubResource = "attach"
)

// execStreamWithTransport executes a kubectl exec with given transport round tripper and upgrader
func (client *client) execStreamWithTransport(ctx context.Context, options *ExecStreamOptions) error {
	var (
		t             term.TTY
		sizeQueue     remotecommand.TerminalSizeQueue
		streamOptions remotecommand.StreamOptions
		tty           = options.TTY
	)

	if options.SubResource == "" {
		options.SubResource = SubResourceExec
	}

	wrapper, upgradeRoundTripper, err := GetUpgraderWrapper(client)
	if err != nil {
		return err
	}

	execRequest := client.KubeClient().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.Pod.Name).
		Namespace(options.Pod.Namespace).
		SubResource(string(options.SubResource))

	if tty {
		tty, t = terminal.SetupTTY(options.Stdin, options.Stdout)
		if options.ForceTTY || tty {
			tty = true
			if t.Raw && options.TerminalSizeQueue == nil {
				// this call spawns a goroutine to monitor/update the terminal size
				sizeQueue = t.MonitorSize(t.GetSize())
			} else if options.TerminalSizeQueue != nil {
				sizeQueue = options.TerminalSizeQueue
				t.Raw = true
			}

			// unset options.Stderr if it was previously set because both stdout and stderr
			// go over t.Out when tty is true
			options.Stderr = nil
			streamOptions = remotecommand.StreamOptions{
				Stdin:             t.In,
				Stdout:            t.Out,
				Tty:               t.Raw,
				TerminalSizeQueue: sizeQueue,
			}
		}
	}
	if !tty {
		streamOptions = remotecommand.StreamOptions{
			Stdin:  options.Stdin,
			Stdout: options.Stdout,
			Stderr: options.Stderr,
		}
	}

	if options.SubResource == SubResourceExec {
		execRequest.VersionedParams(&corev1.PodExecOptions{
			Container: options.Container,
			Command:   options.Command,
			Stdin:     options.Stdin != nil,
			Stdout:    options.Stdout != nil,
			Stderr:    options.Stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)
	} else if options.SubResource == SubResourceAttach {
		execRequest.VersionedParams(&corev1.PodExecOptions{
			Container: options.Container,
			Stdin:     options.Stdin != nil,
			Stdout:    options.Stdout != nil,
			Stderr:    options.Stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)
	}

	exec, err := remotecommand.NewSPDYExecutorForTransports(wrapper, upgradeRoundTripper, "POST", execRequest.URL())
	if err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		errChan <- t.Safe(func() error {
			return exec.Stream(streamOptions)
		})
	}()

	select {
	case <-ctx.Done():
		upgradeRoundTripper.Close()
		<-errChan
		return nil
	case err = <-errChan:
		return err
	}
}

// ExecStreamOptions are the options for ExecStream
type ExecStreamOptions struct {
	Pod *corev1.Pod

	Container string
	Command   []string

	ForceTTY          bool
	TTY               bool
	TerminalSizeQueue remotecommand.TerminalSizeQueue

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	SubResource SubResource
}

// ExecStream executes a command and streams the output to the given streams
func (client *client) ExecStream(ctx context.Context, options *ExecStreamOptions) error {
	return client.execStreamWithTransport(ctx, options)
}

// ExecBuffered executes a command for kubernetes and returns the output and error buffers
func (client *client) ExecBuffered(ctx context.Context, pod *corev1.Pod, container string, command []string, stdin io.Reader) ([]byte, []byte, error) {
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	kubeExecError := client.ExecStream(ctx, &ExecStreamOptions{
		Pod:       pod,
		Container: container,
		Command:   command,
		Stdin:     stdin,
		Stdout:    stdoutBuffer,
		Stderr:    stderrBuffer,
	})
	if kubeExecError != nil {
		if _, ok := kubeExecError.(kubectlExec.CodeExitError); !ok {
			return nil, nil, kubeExecError
		}
	}

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), kubeExecError
}

// ExecBufferedCombined starts a new exec request, waits for it to finish and returns the output to the caller
func (client *client) ExecBufferedCombined(ctx context.Context, pod *corev1.Pod, container string, command []string, stdin io.Reader) ([]byte, error) {
	stdoutBuffer := &bytes.Buffer{}
	kubeExecError := client.ExecStream(ctx, &ExecStreamOptions{
		Pod:       pod,
		Container: container,
		Command:   command,
		Stdin:     stdin,
		Stdout:    stdoutBuffer,
		Stderr:    stdoutBuffer,
	})
	if kubeExecError != nil {
		if _, ok := kubeExecError.(kubectlExec.CodeExitError); !ok {
			return nil, kubeExecError
		}
	}

	return stdoutBuffer.Bytes(), kubeExecError
}
