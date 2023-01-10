package terminal

import (
	"bytes"
	"context"
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/devspace/devpod"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo/v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var _ = DevSpaceDescribe("terminal", func() {
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var (
		f          factory.Factory
		kubeClient *kube.KubeHelper
	)

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()

		kubeClient, err = kube.NewKubeHelper()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should attach", func() {
		tempDir, err := framework.CopyToTempDir("tests/terminal/testdata/attach")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("attach")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		buffer := &bytes.Buffer{}
		devpod.DefaultTerminalStdout = buffer
		devpod.DefaultTerminalStderr = buffer
		devpod.DefaultTerminalStdin = strings.NewReader(`mkdir -p /test/devspace
echo "Hello World!" > /test/devspace/test.txt
sleep 1000000
`)
		defer func() {
			devpod.DefaultTerminalStdout = os.Stdout
			devpod.DefaultTerminalStderr = os.Stderr
			devpod.DefaultTerminalStdin = os.Stdin
		}()

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
				},
				Pipeline: "dev",
				Ctx:      cancelCtx,
			}
			err := devCmd.RunDefault(f)
			if err != nil {
				f.GetLog().Errorf("error: %v", err)
			}
			done <- err
		}()

		// check if file is there
		framework.ExpectRemoteFileContents("ubuntu", ns, "/test/devspace/test.txt", "Hello World!\n")

		cancel()
		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("should restart terminal", func() {
		tempDir, err := framework.CopyToTempDir("tests/terminal/testdata/restart")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("terminal")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		buffer := &bytes.Buffer{}
		devpod.DefaultTerminalStdout = buffer
		devpod.DefaultTerminalStderr = buffer
		devpod.DefaultTerminalStdin = nil
		defer func() {
			devpod.DefaultTerminalStdout = os.Stdout
			devpod.DefaultTerminalStderr = os.Stderr
			devpod.DefaultTerminalStdin = os.Stdin
		}()

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
				},
				Pipeline: "dev",
				Ctx:      cancelCtx,
			}
			done <- devCmd.RunDefault(f)
		}()

		// wait until we get the first hostnames
		var podName string
		err = wait.PollImmediate(time.Second, time.Minute*3, func() (done bool, err error) {
			lines := strings.Split(buffer.String(), "\n")
			if len(lines) <= 1 {
				return false, nil
			}

			podName = strings.TrimSpace(lines[0])
			return true, nil
		})
		framework.ExpectNoError(err)

		// make sure the pod exists
		err = kubeClient.RawClient().CoreV1().Pods(ns).Delete(context.TODO(), podName, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		// wait until pod is terminated
		err = wait.PollImmediate(time.Second, time.Minute*3, func() (done bool, err error) {
			_, err = kubeClient.RawClient().CoreV1().Pods(ns).Get(context.TODO(), podName, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return true, nil
				}

				return false, err
			}

			return false, nil
		})
		framework.ExpectNoError(err)

		// get new pod name
		err = wait.PollImmediate(time.Second, time.Minute*3, func() (done bool, err error) {
			lines := strings.Split(buffer.String(), "\n")
			if len(lines) <= 1 {
				return false, nil
			}

			newPodName := strings.TrimSpace(lines[len(lines)-2])
			if newPodName != podName {
				podName = newPodName
				return true, nil
			}

			return false, nil
		})
		framework.ExpectNoError(err)

		// make sure the pod exists
		_, err = kubeClient.RawClient().CoreV1().Pods(ns).Get(context.TODO(), podName, metav1.GetOptions{})
		framework.ExpectNoError(err)

		// make sure command terminates correctly
		cancel()
		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("should run command locally", func() {
		tempDir, err := framework.CopyToTempDir("tests/terminal/testdata/run_cmd_locally")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("terminal")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "dev",
			Ctx:      cancelCtx,
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)
		framework.ExpectLocalFileContentsImmediately(filepath.Join(tempDir, "terminal-done.txt"), "Hello World!\n")
	})
})

// Buffer is a goroutine safe bytes.Buffer
type Buffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write appends the contents of p to the buffer, growing the buffer as needed. It returns
// the number of bytes written.
func (s *Buffer) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Write(p)
}

// String returns the contents of the unread portion of the buffer
// as a string.  If the Buffer is a nil pointer, it returns "<nil>".
func (s *Buffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.String()
}
