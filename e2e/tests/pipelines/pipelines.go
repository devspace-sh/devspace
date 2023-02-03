package pipelines

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/onsi/ginkgo/v2"
	"os"
	"time"

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
		framework.ExpectLocalFileContentsImmediately("other-profile.txt", "profile1\n")
		framework.ExpectLocalFileContentsImmediately("dep1-test.txt", "test\n")
		framework.ExpectLocalFileContentsImmediately("dep1-test2.txt", "true\n")
		framework.ExpectLocalFileContentsImmediately("dep1-dev-profile.txt", "profile1\n")
		framework.ExpectLocalFileContentsImmediately("dep1-other.txt", "test\n")
		framework.ExpectLocalFileContentsImmediately("dep1-other2.txt", "false\n")
		framework.ExpectLocalFileContentsImmediately("dep1-other3.txt", "false\n")
		framework.ExpectLocalFileContentsImmediately("dep1-other-profile.txt", "profile1\n")
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
})
