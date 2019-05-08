package kubectl

import (
	"io"
	"net/http"

	"github.com/devspace-cloud/devspace/pkg/util/terminal"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	k8sapi "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/kubectl/util/term"
)

// AttachStreamWithTransport attaches to a certain container
func AttachStreamWithTransport(transport http.RoundTripper, upgrader spdy.Upgrader, client *kubernetes.Clientset, pod *k8sv1.Pod, container string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	var t term.TTY
	var sizeQueue remotecommand.TerminalSizeQueue
	var streamOptions remotecommand.StreamOptions

	attachRequest := client.Core().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("attach")

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

	attachRequest.VersionedParams(&k8sapi.PodAttachOptions{
		Container: container,
		Stdin:     stdin != nil,
		Stdout:    stdout != nil,
		Stderr:    stderr != nil,
		TTY:       t.Raw,
	}, legacyscheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutorForTransports(transport, upgrader, "POST", attachRequest.URL())
	if err != nil {
		return err
	}

	return t.Safe(func() error {
		return exec.Stream(streamOptions)
	})
}

// AttachStream attaches to a container in a certain pod
func AttachStream(client *kubernetes.Clientset, pod *k8sv1.Pod, container string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	kubeconfig, err := GetClientConfig()
	if err != nil {
		return err
	}

	wrapper, upgradeRoundTripper, err := spdy.RoundTripperFor(kubeconfig)
	if err != nil {
		return err
	}

	return AttachStreamWithTransport(wrapper, upgradeRoundTripper, client, pod, container, tty, stdin, stdout, stderr)
}
