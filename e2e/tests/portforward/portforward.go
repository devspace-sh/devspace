package portforward

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo/v2"
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

		nginxResp := "<!DOCTYPE html>\n<html>\n<head>\n<title>Welcome to nginx!</title>\n<style>\nhtml { color-scheme: light dark; }\nbody { width: 35em; margin: 0 auto;\nfont-family: Tahoma, Verdana, Arial, sans-serif; }\n</style>\n</head>\n<body>\n<h1>Welcome to nginx!</h1>\n<p>If you see this page, the nginx web server is successfully installed and\nworking. Further configuration is required.</p>\n\n<p>For online documentation and support please refer to\n<a href=\"http://nginx.org/\">nginx.org</a>.<br/>\nCommercial support is available at\n<a href=\"http://nginx.com/\">nginx.com</a>.</p>\n\n<p><em>Thank you for using nginx.</em></p>\n</body>\n</html>"
		framework.ExpectRemoteCurlContents("nginx", ns, "localhost", nginxResp)
		framework.ExpectLocalCurlContents("http://localhost:3000", nginxResp)

		httpServerResp := "Hello World!"
		framework.ExpectLocalCurlContents("http://localhost:8888", httpServerResp)
		framework.ExpectRemoteCurlContents("nginx", ns, "localhost:8888", httpServerResp)

		cancel()
		err = <-done
		framework.ExpectNoError(err)
	})

})
