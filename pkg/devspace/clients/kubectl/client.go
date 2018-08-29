package kubectl

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/covexo/devspace/pkg/devspace/config"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/log"
	dockerterm "github.com/docker/docker/pkg/term"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	k8sapi "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/kubectl/util/term"
)

var privateConfig = &v1.PrivateConfig{}

//NewClient creates a new kubernetes client
func NewClient() (*kubernetes.Clientset, error) {
	config, err := GetClientConfig()

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

//GetClientConfig loads the configuration for kubernetes clients and parses it to *rest.Config
func GetClientConfig() (*rest.Config, error) {
	config.LoadConfig(privateConfig)

	if privateConfig.Cluster == nil {
		return nil, errors.New("Couldn't load cluster config, did you run devspace init")
	}

	return &rest.Config{
		Host:     privateConfig.Cluster.ApiServer,
		Username: privateConfig.Cluster.User.Username,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   []byte(privateConfig.Cluster.CaCert),
			CertData: []byte(privateConfig.Cluster.User.ClientCert),
			KeyData:  []byte(privateConfig.Cluster.User.ClientKey),
		},
	}, nil
}

// ForwardPorts forwards the specified ports from the cluster to the local machine
func ForwardPorts(kubectlClient *kubernetes.Clientset, pod *k8sv1.Pod, ports []string, stopChan chan struct{}, readyChan chan struct{}) error {
	config, err := GetClientConfig()

	if err != nil {
		return err
	}

	execRequest := kubectlClient.Core().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(config)

	if err != nil {
		return err
	}

	logFile := log.GetFileLogger("portforwarding")
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", execRequest.URL())
	fw, err := portforward.New(dialer, ports, stopChan, readyChan, logFile.GetStream(), logFile.GetStream())

	if err != nil {
		return err
	}

	return fw.ForwardPorts()
}

//Exec executes a command for kubectl
func Exec(kubectlClient *kubernetes.Clientset, pod *k8sv1.Pod, container string, command []string, tty bool, errorChannel chan<- error) (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	var t term.TTY

	kubeconfig, err := GetClientConfig()

	if err != nil {
		return nil, nil, nil, err
	}

	execRequest := kubectlClient.Core().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	if tty {
		t = setupTTY()
	}

	execRequest.VersionedParams(&k8sapi.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       t.Raw,
	}, legacyscheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(kubeconfig, "POST", execRequest.URL())

	if err != nil {
		return nil, nil, nil, err
	}

	if tty {
		var sizeQueue remotecommand.TerminalSizeQueue

		if t.Raw {
			// this call spawns a goroutine to monitor/update the terminal size
			sizeQueue = t.MonitorSize(t.GetSize())
		}

		fn := func() error {
			return exec.Stream(remotecommand.StreamOptions{
				Stdin:             os.Stdin,
				Stdout:            os.Stdout,
				Stderr:            os.Stderr,
				Tty:               t.Raw,
				TerminalSizeQueue: sizeQueue,
			})
		}

		if err := t.Safe(fn); err != nil {
			return nil, nil, nil, err
		}

		return nil, nil, nil, nil
	}
	stdinReader, stdinWriter, _ := os.Pipe()
	stdoutReader, stdoutWriter, _ := os.Pipe()
	stderrReader, stderrWriter, _ := os.Pipe()

	go func() {
		streamErr := exec.Stream(remotecommand.StreamOptions{
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: stderrWriter,
			Tty:    tty,
		})
		stdinWriter.Close()
		stdoutWriter.Close()
		stderrWriter.Close()

		errorChannel <- streamErr
	}()
	return stdinWriter, stdoutReader, stderrReader, nil
}

func setupTTY() term.TTY {
	t := term.TTY{
		Out: os.Stdout,
	}

	t.In = os.Stdin

	if !t.IsTerminalIn() {
		log.Info("Unable to use a TTY - input is not a terminal or the right kind of file")

		return t
	}

	// if we get to here, the user wants to attach stdin, wants a TTY, and In is a terminal, so we
	// can safely set t.Raw to true
	t.Raw = true

	_, stdout, _ := dockerterm.StdStreams()

	// io.Copy(os.Stdin, stdin)
	// t.In = stdin

	io.Copy(stdout, os.Stdout)
	t.Out = stdout

	return t
}

//ExecBuffered executes a command for kubernetes and returns the output and error buffers
func ExecBuffered(kubectlClient *kubernetes.Clientset, pod *k8sv1.Pod, container string, command []string) ([]byte, []byte, error) {
	_, stdout, stderr, execErr := Exec(kubectlClient, pod, container, command, false, nil)

	if execErr != nil {
		return nil, nil, execErr
	}
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	streamDone := &sync.WaitGroup{}
	streamDone.Add(2)

	go func() {
		io.Copy(stdoutBuffer, stdout)
		streamDone.Done()
	}()

	go func() {
		io.Copy(stderrBuffer, stderr)
		streamDone.Done()
	}()
	streamDone.Wait()

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
}
