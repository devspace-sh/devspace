package replacepods

import (
	"context"
	"github.com/onsi/ginkgo/v2"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/factory"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
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

	ginkgo.It("should inject restart helper and restart container", func() {
		tempDir, err := framework.CopyToTempDir("tests/replacepods/testdata/restart-helper")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("replacepods")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

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
		framework.ExpectRemoteFileContents("alpine", ns, "/app/test2.txt", "Hello World 123")

		// check if file is there
		framework.ExpectRemoteFileContents("alpine", ns, "/test.txt", "Hello World\n")

		// upload a file and restart the container
		err = os.WriteFile("test1.txt", []byte("Hello World2!"), 0777)
		framework.ExpectNoError(err)

		// wait for uploaded
		framework.ExpectRemoteFileContents("alpine", ns, "/app/test1.txt", "Hello World2!")

		// wait for restarted
		framework.ExpectRemoteFileContents("alpine", ns, "/test.txt", "Hello World\nHello World\n")

		cancel()
		err = <-done
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
		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// check if replica set exists & pod got replaced correctly
		list, err := kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
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
		fileContents, err := os.ReadFile("devspace.yaml")
		framework.ExpectNoError(err)

		newString := strings.ReplaceAll(string(fileContents), "ubuntu:18.04", "alpine:3.14")
		err = os.WriteFile("devspace.yaml", []byte(newString), 0666)
		framework.ExpectNoError(err)

		// rerun
		devCmd = &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// check if replica set exists & pod got replaced correctly
		list, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
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
		purgeCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "purge",
		}
		err = purgeCmd.RunDefault(f)
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
		list, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
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
		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// check if replica set exists & pod got replaced correctly
		list, err := kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
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
		fileContents, err := os.ReadFile("devspace.yaml")
		framework.ExpectNoError(err)

		newString := strings.ReplaceAll(string(fileContents), "ubuntu:18.04", "alpine:3.14")
		err = os.WriteFile("devspace.yaml", []byte(newString), 0666)
		framework.ExpectNoError(err)

		// rerun
		devCmd = &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// check if replica set exists & pod got replaced correctly
		list, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
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

		// now scale down the devspace deployment and upscale the replaced deployment
		_, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).UpdateScale(context.TODO(), "replace-deployment-devspace", &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "replace-deployment-devspace", Namespace: ns},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 0},
		}, metav1.UpdateOptions{})
		framework.ExpectNoError(err)
		_, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).UpdateScale(context.TODO(), "replace-deployment", &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "replace-deployment", Namespace: ns},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 1},
		}, metav1.UpdateOptions{})
		framework.ExpectNoError(err)

		// rerun the devspace command
		devCmd = &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// make sure the deployments are correctly scaled
		deployment, err := kubeClient.Client().KubeClient().AppsV1().Deployments(ns).Get(context.TODO(), "replace-deployment-devspace", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(*deployment.Spec.Replicas, int32(1))
		deployment, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).Get(context.TODO(), "replace-deployment", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(*deployment.Spec.Replicas, int32(0))

		// now purge the deployment and make sure the replica set is deleted as well
		purgeCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "purge",
		}
		err = purgeCmd.RunDefault(f)
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
		list, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 0)
	})

	ginkgo.It("should replace deployment pod with devImage", func() {
		tempDir, err := framework.CopyToTempDir("tests/replacepods/testdata/dev-image")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("devImage")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command
		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// check if replica set exists & pod got replaced correctly
		list, err := kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
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
		framework.ExpectEqual(pods.Items[0].Spec.Containers[0].Image, "nginx:perl")
	})
})
