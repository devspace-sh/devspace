package pipelines

import (
	"context"
	"io/ioutil"
	"os"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
)

var _ = DevSpaceDescribe("portforward", func() {
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
		err = ioutil.WriteFile("test.txt", []byte("abc.txt"), 0777)
		framework.ExpectNoError(err)
		err = ioutil.WriteFile("test2.txt", []byte("abc123.txt"), 0777)
		framework.ExpectNoError(err)
		time.Sleep(time.Millisecond * 500)
		err = ioutil.WriteFile("test3.txt", []byte("abc456.txt"), 0777)
		framework.ExpectNoError(err)
		err = ioutil.WriteFile("test4.txt", []byte("abc789.txt"), 0777)
		framework.ExpectNoError(err)

		framework.ExpectLocalFileContents("test.yaml", "Hello World\nHello World\n")
		framework.ExpectLocalFileContents("test2.yaml", "Hello World\nHello World\n")

		// make a change to a txt file
		err = ioutil.WriteFile("test4.txt", []byte("abc.txt"), 0777)
		framework.ExpectNoError(err)
		err = ioutil.WriteFile("test5.txt", []byte("abc123.txt"), 0777)
		framework.ExpectNoError(err)
		time.Sleep(time.Millisecond * 500)
		err = ioutil.WriteFile("test6.txt", []byte("abc456.txt"), 0777)
		framework.ExpectNoError(err)
		err = ioutil.WriteFile("test7.txt", []byte("abc789.txt"), 0777)
		framework.ExpectNoError(err)

		framework.ExpectLocalFileContents("test.yaml", "Hello World\nHello World\nHello World\n")
		framework.ExpectLocalFileContents("test2.yaml", "Hello World\nHello World\nHello World\n")

		cancel()
		err = <-done
		if err != nil && err != context.Canceled {
			framework.ExpectNoError(err)
		}
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
