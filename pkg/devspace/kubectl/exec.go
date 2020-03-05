package kubectl

import (
	"bytes"
	"io"
	"net/http"

	"github.com/devspace-cloud/devspace/pkg/util/terminal"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	kubectlExec "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/term"
)

// SubResource specifies with sub resources should be used for the container connection (exec or attach)
type SubResource string

const (
	// SubResourceExec creates a new process in the container and attaches to that
	SubResourceExec SubResource = "exec"

	// SubResourceAttach attaches to the top process of the container
	SubResourceAttach SubResource = "attach"
)

// ExecStreamWithTransport executes a kubectl exec with given transport round tripper and upgrader
func (client *client) ExecStreamWithTransport(transport http.RoundTripper, upgrader spdy.Upgrader, pod *k8sv1.Pod, container string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer, subResource SubResource) error {
	var (
		t             term.TTY
		sizeQueue     remotecommand.TerminalSizeQueue
		streamOptions remotecommand.StreamOptions
	)

	execRequest := client.KubeClient().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource(string(subResource))

	if tty {
		t = terminal.SetupTTY(stdin, stdout)

		if t.Raw {
			// this call spawns a goroutine to monitor/update the terminal size
			sizeQueue = t.MonitorSize(t.GetSize())
		}

		streamOptions = remotecommand.StreamOptions{
			Stdin:             t.In,
			Stdout:            t.Out,
			Stderr:            stderr,
			Tty:               t.Raw,
			TerminalSizeQueue: sizeQueue,
		}
	} else {
		streamOptions = remotecommand.StreamOptions{
			Stdin:  stdin,
			Stdout: stdout,
			Stderr: stderr,
		}
	}

	if subResource == SubResourceExec {
		execRequest.VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     stdin != nil,
			Stdout:    stdout != nil,
			Stderr:    stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)
	} else if subResource == SubResourceAttach {
		execRequest.VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Stdin:     stdin != nil,
			Stdout:    stdout != nil,
			Stderr:    stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)
	}

	exec, err := remotecommand.NewSPDYExecutorForTransports(transport, upgrader, "POST", execRequest.URL())
	if err != nil {
		return err
	}

	return t.Safe(func() error {
		return exec.Stream(streamOptions)
	})
}

// ExecStream executes a command and streams the output to the given streams
func (client *client) ExecStream(pod *k8sv1.Pod, container string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	wrapper, upgradeRoundTripper, err := client.GetUpgraderWrapper()
	if err != nil {
		return err
	}

	return client.ExecStreamWithTransport(wrapper, upgradeRoundTripper, pod, container, command, tty, stdin, stdout, stderr, SubResourceExec)
}

// ExecBuffered executes a command for kubernetes and returns the output and error buffers
func (client *client) ExecBuffered(pod *k8sv1.Pod, container string, command []string, input io.Reader) ([]byte, []byte, error) {
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	kubeExecError := client.ExecStream(pod, container, command, false, input, stdoutBuffer, stderrBuffer)
	if kubeExecError != nil {
		if _, ok := kubeExecError.(kubectlExec.CodeExitError); ok == false {
			return nil, nil, kubeExecError
		}
	}

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), kubeExecError
}
