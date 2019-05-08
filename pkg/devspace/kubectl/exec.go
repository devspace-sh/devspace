package kubectl

import (
	"bytes"
	"io"
	"net/http"
	"os"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/terminal"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	k8sapi "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/kubectl/util/term"
)

// ExecStreamWithTransport executes a kubectl exec with given transport round tripper and upgrader
func ExecStreamWithTransport(transport http.RoundTripper, upgrader spdy.Upgrader, client kubernetes.Interface, pod *k8sv1.Pod, container string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	var t term.TTY
	var sizeQueue remotecommand.TerminalSizeQueue
	var streamOptions remotecommand.StreamOptions

	execRequest := client.Core().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	if tty {
		t = terminal.SetupTTY(stdin, stdout)

		if t.Raw {
			// this call spawns a goroutine to monitor/update the terminal size
			sizeQueue = t.MonitorSize(t.GetSize())
		}

		streamOptions = remotecommand.StreamOptions{
			Stdin:             stdin,
			Stdout:            stdout,
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

	execRequest.VersionedParams(&k8sapi.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     stdin != nil,
		Stdout:    stdout != nil,
		Stderr:    stderr != nil,
		TTY:       t.Raw,
	}, legacyscheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutorForTransports(transport, upgrader, "POST", execRequest.URL())
	if err != nil {
		return err
	}

	return t.Safe(func() error {
		return exec.Stream(streamOptions)
	})
}

// ExecStream executes a command and streams the output to the given streams
func ExecStream(config *latest.Config, client kubernetes.Interface, pod *k8sv1.Pod, container string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	kubeconfig, err := GetClientConfig(config)
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := spdy.RoundTripperFor(kubeconfig)
	if err != nil {
		return err
	}

	return ExecStreamWithTransport(wrapper, upgradeRoundTripper, client, pod, container, command, tty, stdin, stdout, stderr)
}

// ExecBuffered executes a command for kubernetes and returns the output and error buffers
func ExecBuffered(config *latest.Config, kubectlClient kubernetes.Interface, pod *k8sv1.Pod, container string, command []string) ([]byte, []byte, error) {
	stdoutReader, stdoutWriter, _ := os.Pipe()
	stderrReader, stderrWriter, _ := os.Pipe()

	err := ExecStream(config, kubectlClient, pod, container, command, false, nil, stdoutWriter, stderrWriter)
	if err != nil {
		return nil, nil, err
	}

	err = stdoutWriter.Close()
	if err != nil {
		return nil, nil, err
	}

	err = stderrWriter.Close()
	if err != nil {
		return nil, nil, err
	}

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	_, err = stdoutBuffer.ReadFrom(stdoutReader)
	if err != nil {
		return nil, nil, err
	}

	_, err = stderrBuffer.ReadFrom(stderrReader)
	if err != nil {
		return nil, nil, err
	}

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
}
