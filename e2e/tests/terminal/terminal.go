package terminal

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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

	ginkgo.It("should restart terminal", func() {
		tempDir, err := framework.CopyToTempDir("tests/terminal/testdata/restart")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("terminal")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		interrupt := make(chan error)
		stdout := &Buffer{}
		go func() {
			devCmd := &cmd.DevCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
				},
				Interrupt: interrupt,
				Stdout:    stdout,
			}
			done <- devCmd.Run(f, []string{"sh", "-c", "while sleep 1; do echo $HOSTNAME; done"})
		}()

		// wait until we get the first hostnames
		var podName string
		err = wait.PollImmediate(time.Second, time.Minute*3, func() (done bool, err error) {
			lines := strings.Split(stdout.String(), "\n")
			if len(lines) <= 1 {
				return false, nil
			}

			podName = lines[0]
			return true, nil
		})
		framework.ExpectNoError(err)

		// make sure the pod exists
		pod, err := kubeClient.RawClient().CoreV1().Pods(ns).Get(context.TODO(), podName, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(pod.Spec.Containers[0].Image, "ubuntu:18.04")

		// now make a change to the config
		fileContents, err := ioutil.ReadFile("devspace.yaml")
		framework.ExpectNoError(err)
		newString := strings.Replace(string(fileContents), "ubuntu:18.04", "alpine:3.14", -1)
		newString = strings.Replace(newString, "container-0", "container-1", -1)
		err = ioutil.WriteFile("devspace.yaml", []byte(newString), 0666)
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
			lines := strings.Split(stdout.String(), "\n")
			if len(lines) <= 1 {
				return false, nil
			}

			newPodName := lines[len(lines)-2]
			if newPodName != podName {
				podName = newPodName
				return true, nil
			}

			return false, nil
		})
		framework.ExpectNoError(err)

		// make sure the pod exists
		pod, err = kubeClient.RawClient().CoreV1().Pods(ns).Get(context.TODO(), podName, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(pod.Spec.Containers[0].Image, "alpine:3.14")
		framework.ExpectEqual(pod.Spec.Containers[0].Name, "container-1")

		// make sure command terminates correctly
		interrupt <- nil
		err = <-done
		framework.ExpectNoError(err)
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
