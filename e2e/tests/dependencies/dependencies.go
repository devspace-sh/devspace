package dependencies

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/podreplace"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
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

	ginkgo.It("should skip dependencies", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/skip")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		config, dependencies, err := framework.LoadConfigWithOptionsAndResolve(f, "", &loader.ConfigOptions{}, dependency.ResolveOptions{SkipDependencies: []string{"flat"}})
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "flat2")
		framework.ExpectEqual(config.Path(), filepath.Join(tempDir, "devspace.yaml"))
	})

	ginkgo.It("should resolve dependencies with dev configuration and hooks", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/dev-sync")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		_, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "dep1")
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
		_, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
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
		_, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
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
		config, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "activated.yaml"))
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
		_, dependencies, err := framework.LoadConfigWithOptions(f, filepath.Join(tempDir, "activated.yaml"), &loader.ConfigOptions{
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
		_, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "deactivated.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly without profile activation
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "nested")
		framework.ExpectEqual(len(dependencies[0].Config().Config().Deployments), 1)
	})

	ginkgo.It("should throw error when profile, profiles, and profile-parents are used together", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/profiles")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		_, _, err = framework.LoadConfig(f, filepath.Join(tempDir, "validate-error.yaml"))
		framework.ExpectErrorMatch(err, "dependencies[0].profiles and dependencies[0].profile & dependencies[0].profileParents cannot be used together")
	})

	ginkgo.It("should resolve dependencies with dependencies.dev.replacePods", func() {
		tempDir, err := framework.CopyToTempDir("tests/dependencies/testdata/dev-replacepods")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// load it from the regular path first
		_, dependencies, err := framework.LoadConfig(f, filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if dependencies were loaded correctly
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(dependencies[0].Name(), "dep")

		ns, err := kubeClient.CreateNamespace("dep-replacepods")
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
		list, err = kubeClient.Client().KubeClient().AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: podreplace.ReplicaSetLabel + "=true"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 0)
	})
})
