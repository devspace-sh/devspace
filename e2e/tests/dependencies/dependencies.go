package dependencies

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"

	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/onsi/ginkgo"
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
			devCmd := &cmd.DevCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
				},
				Ctx: cancelCtx,
			}
			err := devCmd.Run(f)
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
		err = ioutil.WriteFile(filepath.Join(dependencyPath, "test123.txt"), []byte("test123"), 0777)
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
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
		}

		// run the command
		err = deployCmd.Run(f)
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
		devCmd := &cmd.DevCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Ctx: cancelCtx,
		}
		err = devCmd.Run(f)
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
		purgeCmd := &cmd.PurgeCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			All: true,
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
		list, err = kubeClient.Client().KubeClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.ReplacedLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 0)
	})

	ginkgo.It("should resolve cyclic dependencies", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/cyclic")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		_, dependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "dependency")
	})
})
