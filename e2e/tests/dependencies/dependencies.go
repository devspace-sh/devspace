package dependencies

import (
	"bytes"
	"context"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"

	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = DevSpaceDescribe("dependencies", func() {
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var (
		f          *framework.DefaultFactory
		kubeClient *kube.KubeHelper
	)

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()
		kubeClient, err = kube.NewKubeHelper()
	})

	ginkgo.It("should execute cyclic dependencies correctly", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/cyclic2")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("dependencies")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// create a new dev command and start it
		output := &bytes.Buffer{}
		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "devspace.yaml",
			},
			Pipeline: "dev",
			Log:      log.NewStreamLogger(output, output, logrus.DebugLevel),
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// Expect no multiple dependency warning
		gomega.Expect(output.String()).NotTo(
			gomega.ContainSubstring("Seems like you have multiple dependencies with name"),
		)
	})

	ginkgo.It("should wait for dependencies", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/wait")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("dependencies")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// create a new dev command and start it
		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "devspace.yaml",
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)
		framework.ExpectLocalFileContentsImmediately(filepath.Join(tempDir, "out.txt"), `dep3
dep2dep2wait
`)
	})

	ginkgo.It("should not purge common dependency", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/purge")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("dependencies")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// create a new dev command and start it
		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "project1.yaml",
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// make sure the dependencies are correctly deployed
		deploy, err := kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "my-deployment", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Image, "alpine")

		// check if replica set exists & pod got replaced correctly
		list, err := kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 1)
		framework.ExpectEqual(list.Items[0].Spec.Template.Spec.Containers[0].Command, []string{"sleep"})

		// run second dev command
		devCmd = &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "project2.yaml",
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// make sure the dependencies are correctly deployed
		deploy, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "my-deployment", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Image, "alpine")

		// check if replica set exists & pod got replaced correctly
		list, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 1)
		framework.ExpectEqual(list.Items[0].Spec.Template.Spec.Containers[0].Command, []string{"sleep"})

		// purge project 1
		purgeCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "project1.yaml",
			},
			Pipeline: "purge",
		}
		err = purgeCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// make sure the dependencies are correctly deployed
		deploy, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "my-deployment", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(deploy.Spec.Template.Spec.Containers[0].Image, "alpine")

		// check if replica set exists & pod got replaced correctly
		list, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 1)
		framework.ExpectEqual(list.Items[0].Spec.Template.Spec.Containers[0].Command, []string{"sleep"})

		// purge project 2
		purgeCmd = &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:     true,
				Namespace:  ns,
				ConfigPath: "project2.yaml",
			},
			Pipeline: "purge",
		}
		err = purgeCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// make sure the dependencies are correctly deployed
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "my-deployment", metav1.GetOptions{})
		framework.ExpectError(err)

		// check if replica set exists & pod got replaced correctly
		list, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 0)
	})

	ginkgo.It("should deploy git dependency", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/git")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("dependencies")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
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

		// make sure the dependencies are correctly deployed
		id, err := dependencyutil.GetDependencyID(&latest.SourceConfig{
			Git: "https://github.com/loft-sh/e2e-test-dependency.git",
		})
		framework.ExpectNoError(err)

		// calculate dependency path
		dependencyPath := filepath.Join(dependencyutil.DependencyFolderPath, id)

		// wait until file is there
		framework.ExpectLocalFileContents("imports.txt", "Test-dep-test\n")
		framework.ExpectLocalFileContents(filepath.Join(dependencyPath, "dependency-dev.txt"), "Hello I am dependency\n")
		framework.ExpectLocalFileContents(filepath.Join(dependencyPath, "dependency-deploy.txt"), "Hello I am dependency-deploy\n")
		framework.ExpectLocalFileContents("dependency.txt", "Hello again I am dependency-deploy\n")

		// expect remote file
		framework.ExpectRemoteFileContents("alpine", ns, "/app/test.txt", "dependency123")

		// now check if sync is still working
		err = os.WriteFile(filepath.Join(dependencyPath, "test123.txt"), []byte("test123"), 0777)
		framework.ExpectNoError(err)

		// now check if file gets synced
		framework.ExpectRemoteFileContents("alpine", ns, "/app/test123.txt", "test123")

		cancel()
		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("should skip equal dependencies", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/overlapping")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("dependencies")
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
			Pipeline: "dev",
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// make sure the dependencies are correctly deployed
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "dep1", metav1.GetOptions{})
		framework.ExpectNoError(err)
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "dep2", metav1.GetOptions{})
		framework.ExpectNoError(err)
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "dep3", metav1.GetOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("should skip dependencies", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/skip")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		config, dependencies, err := framework.LoadConfigWithOptionsAndResolve(f, kubeClient.Client(), "", &loader.ConfigOptions{}, dependency.ResolveOptions{SkipDependencies: []string{"flat"}})
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "flat2")
		framework.ExpectEqual(config.Path(), filepath.Join(tempDir, "devspace.yaml"))
	})

	ginkgo.It("should resolve dependencies with local path and nested structure", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/nested")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "nested", nil
		})

		// load it from the regular path first
		_, dependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "nested")
	})

	ginkgo.It("should resolve dependencies with local path and flat structure", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/flat")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "flat", nil
		})

		// load it from the regular path first
		_, dependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "flat")
	})

	ginkgo.It("should resolve dependencies and activate dependency profiles", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/profile-activation")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		config, dependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "activated.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly with profile activation
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "nested")
		framework.ExpectEqual(len(dependencies[0].Config().Config().Deployments), 2)
		framework.ExpectEqual(config.Path(), filepath.Join(tempDir, "activated.yaml"))
	})

	ginkgo.It("should resolve dependencies and deactivate activated dependency profiles with --disable-profile-activation", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/profile-activation")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load activated dependencies with --disable-profile-activation
		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		_, dependencies, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "activated.yaml"), &loader.ConfigOptions{
			DisableProfileActivation: true,
		})
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly with profile activation
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "nested")
		framework.ExpectEqual(len(dependencies[0].Config().Config().Deployments), 1)
	})

	ginkgo.It("should resolve dependencies and deactivate dependency profiles", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/profile-activation")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		_, dependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "deactivated.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly without profile activation
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "nested")
		framework.ExpectEqual(len(dependencies[0].Config().Config().Deployments), 1)
	})

	ginkgo.It("should resolve dependencies with dependencies replacePods", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/dev-replacepods")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		_, dependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "dep")

		ns, err := kubeClient.CreateNamespace("dep-replacepods")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command
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
		cancel()

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
		framework.ExpectEqual(pods.Items[0].Spec.Containers[0].Image, "alpine:latest")

		// now purge the deployment, dependency and make sure the replica set is deleted as well
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

	ginkgo.It("should resolve and deploy cyclic git dependencies", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/cyclic")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("dependencies")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		// load it from the regular path first
		_, dependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "dependency")

		// create a new deploy command
		output := &bytes.Buffer{}
		deployCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "deploy",
			Log:      log.NewStreamLogger(output, output, logrus.DebugLevel),
		}

		// run the command
		err = deployCmd.RunDefault(f)
		framework.ExpectNoError(err)

		// Expect no multiple dependency warning
		gomega.Expect(output.String()).NotTo(
			gomega.ContainSubstring("Seems like you have multiple dependencies with name"),
		)

		// expect single deployment
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "nginx", metav1.GetOptions{})
		framework.ExpectNoError(err)
		_, err = kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "nginx2", metav1.GetOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("should not resolve disabled dependencies", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/disabled")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("dep")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		os.Setenv("DEP1_DISABLED", "true")
		defer os.Unsetenv("DEP1_DISABLED")
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
	})

	ginkgo.It("should not run disabled dependencies during run_dependencies --all", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/disabled")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("dep")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		os.Setenv("DEP1_DISABLED", "true")
		defer os.Unsetenv("DEP1_DISABLED")
		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "devspace-all.yaml",
				NoWarn:     true,
				Namespace:  ns,
			},
			Pipeline: "dev",
			Ctx:      cancelCtx,
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)
	})

	ginkgo.It("should error on disabled dependencies during run_dependencies [NAME]", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/disabled")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("dep")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		os.Setenv("DEP1_DISABLED", "true")
		defer os.Unsetenv("DEP1_DISABLED")
		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "devspace-name.yaml",
				NoWarn:     true,
				Namespace:  ns,
			},
			Pipeline: "dev",
			Ctx:      cancelCtx,
			Log:      log.GetFileLogger("devspace-name"),
		}
		err = devCmd.RunDefault(f)
		framework.ExpectError(err)
		framework.ExpectLocalFileContainSubstringImmediately(".devspace/logs/devspace-name.log", "couldn't find dependency dep1")
	})
})
