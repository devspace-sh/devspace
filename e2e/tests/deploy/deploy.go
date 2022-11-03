package deploy

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/factory"
	v1 "k8s.io/api/core/v1"
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
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/kustomize")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// create a new deploy command
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

		// check if services are there
		service, err := kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "my-service", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(service.Labels["kustomize-app"], "devspace")

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

		// check if services are there
		_, err = kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "my-service", metav1.GetOptions{})
		framework.ExpectError(err)
	})

	ginkgo.It("should deploy tanka application", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/tanka")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// create a new deploy command
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

		// check if services are there
		deployment, err := kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "nginx", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectHaveKey(deployment.Labels, "tanka.dev/environment")
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

		// check if services are there
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "nginx", metav1.GetOptions{})
		framework.ExpectError(err)
	})

	ginkgo.It("should deploy multiple namespaces", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/different_namespaces")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("deploy")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		ns2, err := kubeClient.CreateNamespace("deploy")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns2)
			framework.ExpectNoError(err)
		}()

		// exchange kube manifests
		manifests := filepath.Join(tempDir, "kube", "service1.yaml")
		out, err := os.ReadFile(manifests)
		framework.ExpectNoError(err)

		data := strings.ReplaceAll(string(out), "###NAMESPACE1###", ns)
		data = strings.ReplaceAll(data, "###NAMESPACE2###", ns2)

		err = os.WriteFile(manifests, []byte(data), 0777)
		framework.ExpectNoError(err)

		// create a new deploy command
		deployCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
				Vars: []string{
					"NAMESPACE1=" + ns,
					"NAMESPACE2=" + ns2,
				},
			},
			Pipeline: "deploy",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// check if services are there
		_, err = kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "service1", metav1.GetOptions{})
		framework.ExpectNoError(err)
		_, err = kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "service2", metav1.GetOptions{})
		framework.ExpectNoError(err)
		_, err = kubeClient.RawClient().CoreV1().Services(ns2).Get(context.TODO(), "service1", metav1.GetOptions{})
		framework.ExpectNoError(err)
		_, err = kubeClient.RawClient().CoreV1().Services(ns2).Get(context.TODO(), "service2", metav1.GetOptions{})
		framework.ExpectNoError(err)

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

		// check if services are there
		_, err = kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "service1", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "service2", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = kubeClient.RawClient().CoreV1().Services(ns2).Get(context.TODO(), "service1", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = kubeClient.RawClient().CoreV1().Services(ns2).Get(context.TODO(), "service2", metav1.GetOptions{})
		framework.ExpectError(err)
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

		// check if remote cache was deleted
		client, err := kubectl.NewClientFromContext(kubeClient.Client().CurrentContext(), ns, false, kubeClient.Client().KubeConfigLoader())
		framework.ExpectNoError(err)
		config, _, err := framework.LoadConfig(f, client, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)
		_, ok := config.RemoteCache().GetDeployment("test1")
		framework.ExpectEqual(ok, false)
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

	ginkgo.It("should deploy helm application with local source config name", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm-local-source")
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
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: filepath.Join(tempDir, "devspace-name.yaml"),
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

	ginkgo.It("should deploy helm application with local source config path", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/helm-local-source")
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
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: filepath.Join(tempDir, "devspace-path.yaml"),
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

	ginkgo.It("should deploy kubectl application with inline manifest", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/kubectl_inline_manifest")
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
		out, err := kubeClient.ExecByImageSelector("busybox", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")
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

	//nolint:dupl
	ginkgo.It("should deploy kubectl application with patches", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/kubectl_patches")
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

		// check if services are there
		service, err := kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "nginx-deployment", metav1.GetOptions{})
		framework.ExpectNoError(err)
		// check if patches are correctly applied to service
		framework.ExpectEqual(service.Labels["test"], "test234")
		framework.ExpectEqual(service.Spec.Ports[0].Port, int32(8080))

		// check that container is correctly deployed
		out, err := kubeClient.ExecByImageSelector("nginx", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")

		deployment, err := kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "nginx-deployment", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[0].Name, "nginx")
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[0].Image, "nginx:1.23.3")
		framework.ExpectEqual(deployment.Spec.Template.GetObjectMeta().GetLabels(), map[string]string{"app": "nginx", "test": "test123"})
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[1].Name, "busybox")
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[2].Name, "alpine")
		// Ensure the wildcard works
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[0].Env[0], v1.EnvVar{Name: "test", Value: "test123"})
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[1].Env[0], v1.EnvVar{Name: "test", Value: "test123"})
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[2].Env[0], v1.EnvVar{Name: "test", Value: "test123"})
	})

	//nolint:dupl
	ginkgo.It("should deploy kubectl inline manifest application with patches", func() {
		tempDir, err := framework.CopyToTempDir("tests/deploy/testdata/kubectl_inline_manifest_patches")
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

		// check if services are there
		service, err := kubeClient.RawClient().CoreV1().Services(ns).Get(context.TODO(), "nginx-inline-deployment", metav1.GetOptions{})
		framework.ExpectNoError(err)
		// check if patches are correctly applied to service
		framework.ExpectEqual(service.Labels["test"], "test234")
		framework.ExpectEqual(service.Spec.Ports[0].Port, int32(8080))

		// check that container is correctly deployed
		out, err := kubeClient.ExecByImageSelector("nginx", ns, []string{"echo", "-n", "test"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "test")

		deployment, err := kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "nginx-inline-deployment", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[0].Name, "nginx")
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[0].Image, "nginx:1.23.3")
		framework.ExpectEqual(deployment.Spec.Template.GetObjectMeta().GetLabels(), map[string]string{"app": "nginx", "test": "test123"})
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[1].Name, "busybox")
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[2].Name, "alpine")
		// Ensure the wildcard works
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[0].Env[0], v1.EnvVar{Name: "test", Value: "test123"})
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[1].Env[0], v1.EnvVar{Name: "test", Value: "test123"})
		framework.ExpectEqual(deployment.Spec.Template.Spec.Containers[2].Env[0], v1.EnvVar{Name: "test", Value: "test123"})
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
