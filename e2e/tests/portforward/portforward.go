package portforward

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	"net/http"
	"os"
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

		go func() {
			http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				_, _ = fmt.Fprintf(w, "Hello World!")
			})
			fmt.Println("Server started at port 8888")
			err := http.ListenAndServe(":8888", nil)
			framework.ExpectNoError(err)
		}()
	})

	ginkgo.It("should forward and reverse forward ports", func() {
		tempDir, err := framework.CopyToTempDir("tests/portforward/testdata/portforward-simple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("portforward")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command and start it
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

		framework.ExpectRemoteCurlContents("ghcr.io/vaikunthawasi/portforward-ex", ns, "localhost:3000", "Hello World!")
		framework.ExpectLocalCurlContents("http://localhost:3000", "Hello World!")

		framework.ExpectLocalCurlContents("http://localhost:8888", "Hello World!")
		framework.ExpectRemoteCurlContents("ghcr.io/vaikunthawasi/portforward-ex", ns, "localhost:8888", "Hello World!")

		cancel()
		err = <-done
		framework.ExpectNoError(err)
	})

})
