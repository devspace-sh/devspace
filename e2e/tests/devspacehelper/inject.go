package devspacehelper

import (
	"context"
	"github.com/onsi/ginkgo/v2"
	"os"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/devspace/services/inject"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = DevSpaceDescribe("devspacehelper", func() {
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

	ginkgo.It("should download devspacehelper in container using curl", func() {
		tempDir, err := framework.CopyToTempDir("tests/devspacehelper/testdata/curl")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("devspacehelper")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// create a new dev command
		deployCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			SkipPush: true,
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		var pods *corev1.PodList
		err = wait.Poll(time.Second, time.Minute, func() (done bool, err error) {
			pods, err = kubeClient.RawClient().CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=curl-container"})
			if err != nil {
				return false, err
			} else if len(pods.Items) == 0 || len(pods.Items[0].Status.ContainerStatuses) == 0 {
				return false, nil
			}
			return pods.Items[0].Status.ContainerStatuses[0].Ready, nil
		})
		framework.ExpectNoError(err)

		log := log.GetInstance()
		err = inject.InjectDevSpaceHelper(context.TODO(), kubeClient.Client(), &pods.Items[0], "container-0", "", log)
		framework.ExpectNoError(err)

		out, err := kubeClient.ExecByContainer("app=curl-container", "container-0", ns, []string{"ls", inject.DevSpaceHelperContainerPath})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, inject.DevSpaceHelperContainerPath+"\n")

		_, err = kubeClient.ExecByContainer("app=curl-container", "container-0", ns, []string{inject.DevSpaceHelperContainerPath, "version"})
		framework.ExpectNoError(err)
	})

	ginkgo.It("should inject devspacehelper in container via uploading it", func() {
		tempDir, err := framework.CopyToTempDir("tests/devspacehelper/testdata/upload")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("devspacehelper")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// create a new dev command
		deployCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			SkipPush: true,
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		var pods *corev1.PodList
		err = wait.Poll(time.Second, time.Minute, func() (done bool, err error) {
			pods, err = kubeClient.RawClient().CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=non-curl-container"})
			if err != nil {
				return false, err
			}
			return pods.Items[0].Status.ContainerStatuses[0].Ready, nil
		})
		framework.ExpectNoError(err)

		log := log.GetInstance()
		err = inject.InjectDevSpaceHelper(context.TODO(), kubeClient.Client(), &pods.Items[0], "container-0", "", log)
		framework.ExpectNoError(err)

		out, err := kubeClient.ExecByContainer("app=non-curl-container", "container-0", ns, []string{"ls", inject.DevSpaceHelperContainerPath})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, inject.DevSpaceHelperContainerPath+"\n")

		_, err = kubeClient.ExecByContainer("app=non-curl-container", "container-0", ns, []string{inject.DevSpaceHelperContainerPath, "version"})
		framework.ExpectNoError(err)
	})
})
