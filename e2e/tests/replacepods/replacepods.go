package replacepods

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = DevSpaceDescribe("replacepods", func() {
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

	ginkgo.It("should replace statefulset pod", func() {
		tempDir, err := framework.CopyToTempDir("tests/replacepods/testdata/statefulset")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("replacepods")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command
		devCmd := &cmd.DevCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Portforwarding: true,
			Sync:           true,
		}
		err = devCmd.Run(f, []string{"sh", "-c", "exit"})
		framework.ExpectNoError(err)

		// check if replica set exists & pod got replaced correctly
		list, err := kubeClient.Client().KubeClient().AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: podreplace.ReplicaSetLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 1)

		// wait until a pod has started
		var pods *corev1.PodList
		err = wait.Poll(time.Second, time.Minute*3, func() (done bool, err error) {
			pods, err = kubeClient.RawClient().CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel})
			if err != nil {
				return false, err
			}

			return len(pods.Items) == 1, nil
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(pods.Items[0].Spec.Containers[0].Image, "ubuntu:18.04")

		// make sure hostname is correct
		out, err := kubeClient.ExecByContainer("app=nginx", "nginx", ns, []string{"sh", "-c", "echo -n $HOSTNAME"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test-statefulset-0")

		// now make a change to the config
		fileContents, err := ioutil.ReadFile("devspace.yaml")
		framework.ExpectNoError(err)

		newString := strings.Replace(string(fileContents), "ubuntu:18.04", "alpine:3.14", -1)
		err = ioutil.WriteFile("devspace.yaml", []byte(newString), 0666)
		framework.ExpectNoError(err)

		// rerun
		devCmd = &cmd.DevCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Portforwarding: true,
			Sync:           true,
		}
		err = devCmd.Run(f, []string{"sh", "-c", "exit"})
		framework.ExpectNoError(err)

		// check if replica set exists & pod got replaced correctly
		list, err = kubeClient.Client().KubeClient().AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: podreplace.ReplicaSetLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 1)

		// wait until a pod has started
		err = wait.Poll(time.Second, time.Minute*3, func() (done bool, err error) {
			pods, err = kubeClient.RawClient().CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel})
			if err != nil {
				return false, err
			}

			return len(pods.Items) == 1, nil
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(pods.Items[0].Spec.Containers[0].Image, "alpine:3.14")

		// now purge the deployment and make sure the replica set is deleted as well
		purgeCmd := &cmd.PurgeCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
		}
		err = purgeCmd.Run(f)
		framework.ExpectNoError(err)

		// wait until all pods are killed
		err = wait.Poll(time.Second, time.Minute*3, func() (done bool, err error) {
			pods, err = kubeClient.RawClient().CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel})
			if err != nil {
				return false, err
			}

			return len(pods.Items) == 0, nil
		})
		framework.ExpectNoError(err)

		// make sure no replaced replica set exists anymore
		list, err = kubeClient.Client().KubeClient().AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: podreplace.ReplicaSetLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 0)
	})

	ginkgo.It("should replace deployment pod", func() {
		tempDir, err := framework.CopyToTempDir("tests/replacepods/testdata/deployment")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("replacepods")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command
		devCmd := &cmd.DevCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Portforwarding: true,
			Sync:           true,
		}
		err = devCmd.Run(f, []string{"sh", "-c", "exit"})
		framework.ExpectNoError(err)

		// check if replica set exists & pod got replaced correctly
		list, err := kubeClient.Client().KubeClient().AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: podreplace.ReplicaSetLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 1)

		// wait until a pod has started
		var pods *corev1.PodList
		err = wait.Poll(time.Second, time.Minute, func() (done bool, err error) {
			pods, err = kubeClient.RawClient().CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel})
			if err != nil {
				return false, err
			}

			return len(pods.Items) == 1, nil
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(pods.Items[0].Spec.Containers[0].Image, "ubuntu:18.04")

		// now make a change to the config
		fileContents, err := ioutil.ReadFile("devspace.yaml")
		framework.ExpectNoError(err)

		newString := strings.Replace(string(fileContents), "ubuntu:18.04", "alpine:3.14", -1)
		err = ioutil.WriteFile("devspace.yaml", []byte(newString), 0666)
		framework.ExpectNoError(err)

		// rerun
		devCmd = &cmd.DevCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Portforwarding: true,
			Sync:           true,
		}
		err = devCmd.Run(f, []string{"sh", "-c", "exit"})
		framework.ExpectNoError(err)

		// check if replica set exists & pod got replaced correctly
		list, err = kubeClient.Client().KubeClient().AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: podreplace.ReplicaSetLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 1)

		// wait until a pod has started
		err = wait.Poll(time.Second, time.Minute, func() (done bool, err error) {
			pods, err = kubeClient.RawClient().CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel})
			if err != nil {
				return false, err
			}

			return len(pods.Items) == 1, nil
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(pods.Items[0].Spec.Containers[0].Image, "alpine:3.14")

		// now purge the deployment and make sure the replica set is deleted as well
		purgeCmd := &cmd.PurgeCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
		}
		err = purgeCmd.Run(f)
		framework.ExpectNoError(err)

		// wait until all pods are killed
		err = wait.Poll(time.Second, time.Minute, func() (done bool, err error) {
			pods, err = kubeClient.RawClient().CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel})
			if err != nil {
				return false, err
			}

			return len(pods.Items) == 0, nil
		})
		framework.ExpectNoError(err)

		// make sure no replaced replica set exists anymore
		list, err = kubeClient.Client().KubeClient().AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: podreplace.ReplicaSetLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 0)
	})
})
