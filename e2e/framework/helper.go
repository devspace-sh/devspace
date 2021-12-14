package framework

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ExpectEqual expects the specified two are the same, otherwise an exception raises
func ExpectEqual(actual interface{}, extra interface{}, explain ...interface{}) {
	gomega.ExpectWithOffset(1, actual).To(gomega.Equal(extra), explain...)
}

// ExpectNotEqual expects the specified two are not the same, otherwise an exception raises
func ExpectNotEqual(actual interface{}, extra interface{}, explain ...interface{}) {
	gomega.ExpectWithOffset(1, actual).NotTo(gomega.Equal(extra), explain...)
}

// ExpectError expects an error happens, otherwise an exception raises
func ExpectError(err error, explain ...interface{}) {
	gomega.ExpectWithOffset(1, err).To(gomega.HaveOccurred(), explain...)
}

// ExpectMatchError expects an error happens and has a message matching the given string, otherwise an exception raises
func ExpectErrorMatch(err error, msg string, explain ...interface{}) {
	gomega.ExpectWithOffset(1, err).To(gomega.HaveOccurred(), explain...)
	gomega.ExpectWithOffset(1, err, explain...).To(gomega.MatchError(msg), explain...)
}

// ExpectNoError checks if "err" is set, and if so, fails assertion while logging the error.
func ExpectNoError(err error, explain ...interface{}) {
	ExpectNoErrorWithOffset(1, err, explain...)
}

// ExpectNoErrorWithOffset checks if "err" is set, and if so, fails assertion while logging the error at "offset" levels above its caller
// (for example, for call chain f -> g -> ExpectNoErrorWithOffset(1, ...) error would be logged for "f").
func ExpectNoErrorWithOffset(offset int, err error, explain ...interface{}) {
	gomega.ExpectWithOffset(1+offset, err).NotTo(gomega.HaveOccurred(), explain...)
}

// ExpectConsistOf expects actual contains precisely the extra elements.  The ordering of the elements does not matter.
func ExpectConsistOf(actual interface{}, extra interface{}, explain ...interface{}) {
	gomega.ExpectWithOffset(1, actual).To(gomega.ConsistOf(extra), explain...)
}

// ExpectHaveKey expects the actual map has the key in the keyset
func ExpectHaveKey(actual interface{}, key interface{}, explain ...interface{}) {
	gomega.ExpectWithOffset(1, actual).To(gomega.HaveKey(key), explain...)
}

// ExpectEmpty expects actual is empty
func ExpectEmpty(actual interface{}, explain ...interface{}) {
	gomega.ExpectWithOffset(1, actual).To(gomega.BeEmpty(), explain...)
}

func ExpectRemoteFileContents(imageSelector string, namespace string, filePath string, contents string) {
	kubeClient, err := kube.NewKubeHelper()
	ExpectNoErrorWithOffset(1, err)

	err = wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
		out, err := kubeClient.ExecByImageSelector(imageSelector, namespace, []string{"cat", filePath})
		if err != nil {
			return false, nil
		}

		return out == contents, nil
	})
	ExpectNoErrorWithOffset(1, err)
}

func ExpectRemoteFileNotFound(imageSelector string, namespace string, filePath string) {
	kubeClient, err := kube.NewKubeHelper()
	ExpectNoErrorWithOffset(1, err)

	fileExists := "file exists"
	fileNotFound := "file not found"
	err = wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
		test := []string{"sh", "-c", fmt.Sprintf("test -e %s && echo %s || echo %s", filePath, fileExists, fileNotFound)}
		out, err := kubeClient.ExecByImageSelector(imageSelector, namespace, test)
		if err != nil {
			return false, err
		}

		out = strings.Trim(out, "\n")

		if out == fileExists {
			return false, errors.New("file should not exist")
		}

		return out == fileNotFound, nil
	})
	ExpectNoErrorWithOffset(1, err)
}

func ExpectRemoteContainerFileContents(labelSelector, container string, namespace string, filePath string, contents string) {
	kubeClient, err := kube.NewKubeHelper()
	ExpectNoErrorWithOffset(1, err)

	err = wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
		out, err := kubeClient.ExecByContainer(labelSelector, container, namespace, []string{"cat", filePath})
		if err != nil {
			return false, nil
		}
		return out == contents, nil
	})
	ExpectNoErrorWithOffset(1, err)
}

func ExpectLocalFileContentsImmediately(filePath string, contents string) {
	out, err := ioutil.ReadFile(filePath)
	ExpectNoError(err)
	gomega.ExpectWithOffset(2, string(out)).To(gomega.Equal(contents))
}

func ExpectLocalFileContents(filePath string, contents string) {
	err := wait.PollImmediate(time.Second, time.Minute*2, func() (done bool, err error) {
		out, err := ioutil.ReadFile(filePath)
		if err != nil {
			if !os.IsNotExist(err) {
				return false, err
			}

			return false, nil
		}

		return string(out) == contents, nil
	})
	ExpectNoErrorWithOffset(1, err)
}

func ExpectLocalFileNotFound(filePath string) {
	_, err := os.Stat(filePath)
	gomega.ExpectWithOffset(1, os.IsNotExist(err)).Should(gomega.BeTrue())
}

func ExpectDeleteNamespace(k *kube.KubeHelper, name string) {
	err := k.DeleteNamespace(name)
	ExpectNoError(err)
}
