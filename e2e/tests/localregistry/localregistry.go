package localregistry

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/devspace/build/registry"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

var _ = DevSpaceDescribe("localregistry", func() {

	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	var (
		// create a new factory
		f factory.Factory

		// create logger
		// log logpkg.Logger

		// create kube helper
		kubeClient *kube.KubeHelper

		pollingInterval = time.Second * 2

		pollingDurationLong = time.Second * 30
	)

	// create context
	ctx := context.Background()

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()

		kubeClient, err = kube.NewKubeHelper()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should build dockerfile with docker and use local registry", func() {
		tempDir, err := framework.CopyToTempDir("tests/localregistry/testdata/local-registry")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("localregistry")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			defer ginkgo.GinkgoRecover()

			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
				},
				Pipeline: "dev",
				Ctx:      cancelCtx,
			}

			done <- devCmd.RunDefault(f)
		}()

		var registryHost string
		ginkgo.By("Checking registry for pushed image")
		gomega.Eventually(func() (string, error) {
			service, err := kubeClient.RawClient().CoreV1().Services(ns).Get(ctx, "registry", v1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return "", nil
				}

				return "", err
			}

			registryPort := registry.GetNodePort(service)
			registryHost = fmt.Sprintf("localhost:%d", registryPort)
			registry, err := name.NewRegistry(registryHost)
			if err != nil {
				return "", err
			}

			images, err := remote.Catalog(ctx, registry)
			if err != nil {
				return "", err
			}

			if len(images) == 0 {
				return "", err
			}

			return images[0], nil
		}, pollingDurationLong, pollingInterval).
			Should(gomega.Equal("my-docker-username/helloworld"))

		ginkgo.By("Checking get_image output")
		gomega.Eventually(func() (string, error) {
			out, err := ioutil.ReadFile("get_image.out")
			if err != nil {
				if !os.IsNotExist(err) {
					return "", err
				}

				return "", nil
			}
			return string(out), nil
		}, pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking %{runtime.images.app} output")
		gomega.Eventually(func() (string, error) {
			out, err := ioutil.ReadFile("app.out")
			if err != nil {
				if !os.IsNotExist(err) {
					return "", err
				}

				return "", nil
			}
			return string(out), nil
		}, pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking %{runtime.images.app.image} output")
		gomega.Eventually(func() (string, error) {
			out, err := ioutil.ReadFile("app_image.out")
			if err != nil {
				if !os.IsNotExist(err) {
					return "", err
				}

				return "", nil
			}
			return string(out), nil
		}, pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking deployment container1")
		gomega.Eventually(func() (string, error) {
			deployment, err := kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "app", v1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return "", nil
				}

				return "", err
			}

			for _, container := range deployment.Spec.Template.Spec.Containers {
				if container.Name == "container1" {
					return container.Image, nil
				}
			}

			return "", nil
		}, pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking deployment container2")
		gomega.Eventually(func() (string, error) {
			deployment, err := kubeClient.RawClient().AppsV1().Deployments(ns).Get(context.TODO(), "app", v1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return "", nil
				}

				return "", err
			}

			for _, container := range deployment.Spec.Template.Spec.Containers {
				if container.Name == "container2" {
					return container.Image, nil
				}
			}

			return "", nil
		}, pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("should error when local registry is required and not supported by build type", func() {
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/kaniko")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		output := &bytes.Buffer{}
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			Pipeline: "build",
			SkipPush: true,
			Log:      logpkg.NewStreamLogger(output, output, logrus.DebugLevel),
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectError(err)
		gomega.Expect(output.String()).To(
			gomega.ContainSubstring("unable to push image my-docker-username/helloworld-kaniko and only docker and buildkit builds support using a local registry"),
		)
	})

	ginkgo.It("should error when local registry is configured and not supported by build type", func() {
		tempDir, err := framework.CopyToTempDir("tests/localregistry/testdata/local-registry-invalid")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		output := &bytes.Buffer{}
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			Pipeline: "build",
			Log:      logpkg.NewStreamLogger(output, output, logrus.DebugLevel),
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectError(err)
		gomega.Expect(output.String()).To(
			gomega.ContainSubstring("local registry is configured for this image build, but is only available for docker and buildkit image builds"),
		)
	})

	ginkgo.It("should error when local registry is required and disabled by configuration", func() {
		tempDir, err := framework.CopyToTempDir("tests/localregistry/testdata/local-registry-disabled")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// create build command
		output := &bytes.Buffer{}
		buildCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			Pipeline: "build",
			Log:      logpkg.NewStreamLogger(output, output, logrus.DebugLevel),
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectError(err)
		gomega.Expect(output.String()).To(
			gomega.ContainSubstring("build images: unable to push image my-docker-username/helloworld-kaniko and using a local registry is disabled"),
		)
	})
})
