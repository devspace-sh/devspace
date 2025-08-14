package pipelines

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
)

var _ = DevSpaceDescribe("pipelines", func() {
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

	ginkgo.It("should resolve pipeline flags", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/flags")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		rootCmd := cmd.NewRootCmd(f)
		persistentFlags := rootCmd.PersistentFlags()
		globalFlags := flags.SetGlobalFlags(persistentFlags)
		globalFlags.NoWarn = true
		globalFlags.Namespace = ns
		globalFlags.Profiles = []string{"profile1"}

		cmdCtx := values.WithCommandFlags(context.Background(), globalFlags.Flags)
		cmdCtx = values.WithFlagsMap(cmdCtx, map[string]string{
			"test":  "test",
			"test2": "",
		})

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: globalFlags,
			Pipeline:    "dev",
			Ctx:         cmdCtx,
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		framework.ExpectLocalFileContentsImmediately("test.txt", "test\n")
		framework.ExpectLocalFileContentsImmediately("test2.txt", "\n")
		framework.ExpectLocalFileContentsImmediately("dev-profile.txt", "profile1\n")
		framework.ExpectLocalFileContentsImmediately("other.txt", "test\n")
		framework.ExpectLocalFileContentsImmediately("other2.txt", "false\n")
		framework.ExpectLocalFileContentsImmediately("other3.txt", "true\n")
		framework.ExpectLocalFileContentsImmediately("other4-0.txt", "one\n")
		framework.ExpectLocalFileContentsImmediately("other4-1.txt", "two\n")
		framework.ExpectLocalFileContentsImmediately("other-profile.txt", "profile1\n")
		framework.ExpectLocalFileContentsImmediately("dep1-test.txt", "test\n")
		framework.ExpectLocalFileContentsImmediately("dep1-test2.txt", "true\n")
		framework.ExpectLocalFileContentsImmediately("dep1-dev-profile.txt", "profile1\n")
		framework.ExpectLocalFileContentsImmediately("dep1-other.txt", "test\n")
		framework.ExpectLocalFileContentsImmediately("dep1-other2.txt", "false\n")
		framework.ExpectLocalFileContentsImmediately("dep1-other3.txt", "false\n")
		framework.ExpectLocalFileContentsImmediately("dep1-other-profile.txt", "profile1\n")
	})

	ginkgo.It("should resolve pipeline override array flags", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/flags")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		rootCmd := cmd.NewRootCmd(f)
		persistentFlags := rootCmd.PersistentFlags()
		globalFlags := flags.SetGlobalFlags(persistentFlags)
		globalFlags.NoWarn = true
		globalFlags.Namespace = ns
		globalFlags.Profiles = []string{"profile1"}

		cmdCtx := values.WithCommandFlags(context.Background(), globalFlags.Flags)
		cmdCtx = values.WithFlagsMap(cmdCtx, map[string]string{
			"other":  "test",
			"other2": "false",
			"other3": "true",
			"other4": "three four",
		})

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: globalFlags,
			Pipeline:    "other",
			Ctx:         cmdCtx,
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		framework.ExpectLocalFileContentsImmediately("other.txt", "test\n")
		framework.ExpectLocalFileContentsImmediately("other2.txt", "false\n")
		framework.ExpectLocalFileContentsImmediately("other3.txt", "true\n")
		framework.ExpectLocalFileContentsImmediately("other-profile.txt", "profile1\n")
		framework.ExpectLocalFileContentsImmediately("other4-0.txt", "three\n")
		framework.ExpectLocalFileContentsImmediately("other4-1.txt", "four\n")
	})

	ginkgo.It("should resolve pipeline override with --set-flags", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/flags")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		rootCmd := cmd.NewRootCmd(f)
		persistentFlags := rootCmd.PersistentFlags()
		globalFlags := flags.SetGlobalFlags(persistentFlags)
		globalFlags.NoWarn = true
		globalFlags.Namespace = ns
		globalFlags.Profiles = []string{"profile1"}

		cmdCtx := values.WithCommandFlags(context.Background(), globalFlags.Flags)
		cmdCtx = values.WithFlagsMap(cmdCtx, map[string]string{})

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: globalFlags,
			Pipeline:    "other-override",
			Ctx:         cmdCtx,
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		framework.ExpectLocalFileContentsImmediately("other.txt", "test\n")
		framework.ExpectLocalFileContentsImmediately("other2.txt", "true\n")
		framework.ExpectLocalFileContentsImmediately("other3.txt", "true\n")
		framework.ExpectLocalFileContentsImmediately("other-profile.txt", "profile1\n")
		framework.ExpectLocalFileContentsImmediately("other4-0.txt", "five\n")
		framework.ExpectLocalFileContentsImmediately("other4-1.txt", "six\n")
	})

	ginkgo.It("should resolve dependency pipeline flag defaults", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/flags")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		rootCmd := cmd.NewRootCmd(f)
		persistentFlags := rootCmd.PersistentFlags()
		globalFlags := flags.SetGlobalFlags(persistentFlags)
		globalFlags.NoWarn = true
		globalFlags.Namespace = ns
		globalFlags.Profiles = []string{"profile1"}

		cmdCtx := values.WithCommandFlags(context.Background(), globalFlags.Flags)
		cmdCtx = values.WithFlagsMap(cmdCtx, map[string]string{})

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: globalFlags,
			Pipeline:    "arr-dep1",
			Ctx:         cmdCtx,
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		framework.ExpectLocalFileContentsImmediately("arr-0.txt", "one")
		framework.ExpectLocalFileContentsImmediately("arr-1.txt", "two")
	})

	ginkgo.It("should resolve dependency pipeline flag defaults", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/flags")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		rootCmd := cmd.NewRootCmd(f)
		persistentFlags := rootCmd.PersistentFlags()
		globalFlags := flags.SetGlobalFlags(persistentFlags)
		globalFlags.NoWarn = true
		globalFlags.Namespace = ns
		globalFlags.Profiles = []string{"profile1"}

		cmdCtx := values.WithCommandFlags(context.Background(), globalFlags.Flags)
		cmdCtx = values.WithFlagsMap(cmdCtx, map[string]string{})

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: globalFlags,
			Pipeline:    "arr-dep1-override",
			Ctx:         cmdCtx,
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)

		framework.ExpectLocalFileContentsImmediately("arr-0.txt", "three")
		framework.ExpectLocalFileContentsImmediately("arr-1.txt", "")
	})

	ginkgo.It("should exec container", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/exec_container")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)
		framework.ExpectLocalFileContentsImmediately("test.txt", "Hello World!\n")
	})

	ginkgo.It("should watch files", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/run_watch")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

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

		framework.ExpectLocalFileContents("test.yaml", "Hello World\n")
		framework.ExpectLocalFileContents("test2.yaml", "Hello World\n")

		// make a change to a txt file
		err = os.WriteFile("test.txt", []byte("abc.txt"), 0777)
		framework.ExpectNoError(err)
		err = os.WriteFile("test2.txt", []byte("abc123.txt"), 0777)
		framework.ExpectNoError(err)
		time.Sleep(time.Millisecond * 500)
		err = os.WriteFile("test3.txt", []byte("abc456.txt"), 0777)
		framework.ExpectNoError(err)
		err = os.WriteFile("test4.txt", []byte("abc789.txt"), 0777)
		framework.ExpectNoError(err)

		framework.ExpectLocalFileContents("test.yaml", "Hello World\nHello World\n")
		framework.ExpectLocalFileContents("test2.yaml", "Hello World\nHello World\n")

		// make a change to a txt file
		err = os.WriteFile("test4.txt", []byte("abc.txt"), 0777)
		framework.ExpectNoError(err)
		err = os.WriteFile("test5.txt", []byte("abc123.txt"), 0777)
		framework.ExpectNoError(err)
		time.Sleep(time.Millisecond * 500)
		err = os.WriteFile("test6.txt", []byte("abc456.txt"), 0777)
		framework.ExpectNoError(err)
		err = os.WriteFile("test7.txt", []byte("abc789.txt"), 0777)
		framework.ExpectNoError(err)

		framework.ExpectLocalFileContents("test.yaml", "Hello World\nHello World\nHello World\n")
		framework.ExpectLocalFileContents("test2.yaml", "Hello World\nHello World\nHello World\n")

		cancel()
		err = <-done
		if err != nil && err != context.Canceled {
			framework.ExpectNoError(err)
		}
	})

	ginkgo.It("should watch files with excludes, no ./", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/run_watch")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.ExpectDeleteNamespace, kubeClient, ns)

		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(ctx)
		ginkgo.DeferCleanup(cancel)

		output := &bytes.Buffer{}
		multiWriter := io.MultiWriter(output, os.Stdout)
		log := logpkg.NewStreamLogger(multiWriter, multiWriter, logrus.DebugLevel)

		go func() {
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
				},
				Pipeline: "no-dot-slash-exclude",
				Ctx:      cancelCtx,
				Log:      log,
			}
			err := devCmd.RunDefault(f)
			if err != nil {
				f.GetLog().Errorf("error: %v", err)
			}
			done <- err
		}()

		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(output.String()).Should(gomega.ContainSubstring("Start watching"))
		}).
			WithTimeout(5 * time.Second).
			Should(gomega.Succeed())

		err = os.WriteFile("foo1/test.txt", []byte("abc.txt"), 0777)
		framework.ExpectNoError(err)

		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(output.String()).Should(gomega.ContainSubstring("Restarting command because 'foo1/test.txt' has changed"))
		}).
			WithTimeout(5 * time.Second).
			Should(gomega.Succeed())

		err = os.WriteFile("foo2/test.txt", []byte("abc.txt"), 0777)
		framework.ExpectNoError(err)

		gomega.Consistently(func(g gomega.Gomega) {
			g.Expect(output.String()).ShouldNot(gomega.ContainSubstring("Restarting command because 'foo2/test.txt' has changed"))
		}).
			WithTimeout(5 * time.Second).
			Should(gomega.Succeed())

		cancel()
		err = <-done
		if err != nil && err != context.Canceled {
			framework.ExpectNoError(err)
		}
	})

	ginkgo.It("should watch files with excludes, with ./", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/run_watch")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.ExpectDeleteNamespace, kubeClient, ns)

		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(ctx)
		ginkgo.DeferCleanup(cancel)

		output := &bytes.Buffer{}
		multiWriter := io.MultiWriter(output, os.Stdout)
		log := logpkg.NewStreamLogger(multiWriter, multiWriter, logrus.DebugLevel)

		go func() {
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
				},
				Pipeline: "dot-slash-exclude",
				Ctx:      cancelCtx,
				Log:      log,
			}
			err := devCmd.RunDefault(f)
			if err != nil {
				f.GetLog().Errorf("error: %v", err)
			}
			done <- err
		}()

		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(output.String()).Should(gomega.ContainSubstring("Start watching"))
		}).
			WithTimeout(5 * time.Second).
			Should(gomega.Succeed())

		err = os.WriteFile("foo1/test.txt", []byte("abc.txt"), 0777)
		framework.ExpectNoError(err)

		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(output.String()).Should(gomega.ContainSubstring("Restarting command because 'foo1/test.txt' has changed"))
		}).
			WithTimeout(5 * time.Second).
			Should(gomega.Succeed())

		err = os.WriteFile("foo2/test.txt", []byte("abc.txt"), 0777)
		framework.ExpectNoError(err)

		gomega.Consistently(func(g gomega.Gomega) {
			g.Expect(output.String()).ShouldNot(gomega.ContainSubstring("Restarting command because 'foo2/test.txt' has changed"))
		}).
			WithTimeout(5 * time.Second).
			Should(gomega.Succeed())

		cancel()
		err = <-done
		if err != nil && err != context.Canceled {
			framework.ExpectNoError(err)
		}
	})
	ginkgo.It("should fail to watch files with unquoted globbing", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/run_watch")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.ExpectDeleteNamespace, kubeClient, ns)

		cancelCtx, cancel := context.WithCancel(ctx)
		ginkgo.DeferCleanup(cancel)

		output := &bytes.Buffer{}
		multiWriter := io.MultiWriter(output, os.Stdout)
		log := logpkg.NewStreamLogger(multiWriter, multiWriter, logrus.DebugLevel)

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "unquoted-glob",
			Ctx:      cancelCtx,
			Log:      log,
		}
		err = devCmd.RunDefault(f)
		framework.ExpectError(err)
	})

	ginkgo.It("should use --set and --set-string values from run_pipelines command", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/run_pipelines")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			defer ginkgo.GinkgoRecover()
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:     true,
					Namespace:  ns,
					ConfigPath: "devspace.yaml",
				},
				Pipeline: "dev",
				Ctx:      cancelCtx,
			}
			done <- devCmd.RunDefault(f)
		}()

		// check if deployments are there
		framework.ExpectContainerNameAndImageEqual(ns, "dev", "nginx", "mynginx")

		cancel()
		<-done
	})

	ginkgo.It("should use --set and --set-string values from run_default_pipeline command", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/run_default_pipeline")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			defer ginkgo.GinkgoRecover()
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:     true,
					Namespace:  ns,
					ConfigPath: "devspace.yaml",
				},
				Pipeline: "dev",
				Ctx:      cancelCtx,
			}
			done <- devCmd.RunDefault(f)
		}()

		// check if deployments are there
		framework.ExpectContainerNameAndImageEqual(ns, "dev", "nginx", "mynginx")

		cancel()
		<-done
	})

	ginkgo.It("should get value from config", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/getconfigvalue")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pipelines")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn:    true,
				Namespace: ns,
			},
			Pipeline: "dev",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)
	})

	ginkgo.It("should not panic on inalid kubeconfig", func() {
		tempDir, err := framework.CopyToTempDir("tests/pipelines/testdata/invalid_kubeconfig")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		origEnv := os.Getenv("KUBE_CONFIG")
		defer os.Setenv("KUBE_CONFIG", origEnv)

		os.Setenv("KUBE_CONFIG", "nonexistent.yaml")
		newEnv := os.Getenv("KUBE_CONFIG")
		framework.ExpectEqual(newEnv, "nonexistent.yaml")

		devCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			Pipeline: "deploy",
		}
		err = devCmd.RunDefault(f)
		framework.ExpectNoError(err)
	})
})
