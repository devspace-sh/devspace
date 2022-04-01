package deploy

import (
	"context"
	"os"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = DevSpaceDescribe("deploy", func() {
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

	ginkgo.It("should deploy kustomize application", func() {
		// TODO
	})

	ginkgo.It("should deploy concurrent deployments", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm_concurrent_new")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
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
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// check if deployments are there
		deploy, err := kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "test1", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Name, "test")
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Image, "alpine")
		deploy, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "test2", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Name, "test")
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Image, "alpine")
		deploy, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "test3", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Name, "test")
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Image, "alpine")
		deploy, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "test4", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Name, "test")
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Image, "alpine")
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "base", metav1.GetOptions{})
		framework.ExpectError(err)

		// create a new purge command
		purgeCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "purge",
		}

		// run the command
		err = purgeCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// check if deployments are still there
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "test1", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "test2", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "test3", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "test4", metav1.GetOptions{})
		framework.ExpectError(err)
	})

	ginkgo.It("should deploy helm application", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
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
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		out, err := kubeClient.ExecByImageSelector("nginx", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")
	})

	ginkgo.It("should deploy kubectl application", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/kubectl")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
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
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		out, err := kubeClient.ExecByImageSelector("nginx", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")

		// wait until nginx pod is reachable
		out, err = kubeClient.ExecByImageSelector("busybox", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")

		// check if service is there
		_, err = kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "webserver-simple-service", metav1.GetOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("should deploy helm chart from git repo", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm_git")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
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
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		out, err := kubeClient.ExecByImageSelector("nginx", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")
	})

	ginkgo.It("should deploy helm chart from specific branch in git repo", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm_git_branch")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
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
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		out, err := kubeClient.ExecByImageSelector("nginx", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")
	})

	ginkgo.It("should deploy helm chart from subpath in git repo", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm_git_subpath")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
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
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		out, err := kubeClient.ExecByImageSelector("nginx", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")
	})

	ginkgo.It("should deploy applications concurrently", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm_concurrent_all")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
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
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		out, err := kubeClient.ExecByContainer("app.kubernetes.io/component=test", "container-0", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		out2, err := kubeClient.ExecByContainer("app.kubernetes.io/component=test-2", "container-0", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		out3, err := kubeClient.ExecByContainer("app.kubernetes.io/component=test-3", "container-0", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)

		framework.ExpectEqual(out, "test")
		framework.ExpectEqual(out2, "test")
		framework.ExpectEqual(out3, "test")
	})

	ginkgo.It("should deploy applications mixed concurrently and sequentially", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm_concurrent_sequential")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
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
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// wait until nginx pod is reachable
		out, err := kubeClient.ExecByContainer("app.kubernetes.io/component=test", "container-0", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		out2, err := kubeClient.ExecByContainer("app.kubernetes.io/component=test-2", "container-0", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		out3, err := kubeClient.ExecByContainer("app.kubernetes.io/component=test-3", "container-0", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)

		framework.ExpectEqual(out, "test")
		framework.ExpectEqual(out2, "test")
		framework.ExpectEqual(out3, "test")
	})
})
