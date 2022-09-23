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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	"k8s.io/apimachinery/pkg/api/resource"
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

	ginkgo.It("should build dockerfile with docker and use local registry with helm deployment", func() {
		tempDir, err := framework.CopyToTempDir("tests/localregistry/testdata/local-registry-helm")
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
		ginkgo.By("Waiting for registry service node port")
		gomega.Eventually(func() (*corev1.Service, error) {
			service, err := getRegistryService(ctx, kubeClient, ns)
			if err != nil {
				return nil, err
			}

			if service != nil {
				registryPort := registry.GetServicePort(service)
				if registryPort.NodePort != 0 {
					registryHost = fmt.Sprintf("localhost:%d", registryPort.NodePort)
					return service, nil
				}
			}

			return nil, nil
		}, pollingDurationLong, pollingInterval).
			ShouldNot(gomega.BeNil())

		ginkgo.By("Checking registry for pushed image")
		gomega.Eventually(getImages(ctx, registryHost), pollingDurationLong, pollingInterval).
			Should(gomega.ContainElement("my-docker-username/helloworld"))

		ginkgo.By("Checking deployment container1")
		gomega.Eventually(selectContainerImage(kubeClient, ns, "app", "container1"), pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking deployment container2")
		gomega.Eventually(selectContainerImage(kubeClient, ns, "app", "container2"), pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking deployment container3")
		gomega.Eventually(selectContainerImage(kubeClient, ns, "app", "container3"), pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("should build dockerfile with buildkit and use local registry", func() {
		tempDir, err := framework.CopyToTempDir("tests/localregistry/testdata/local-registry-buildkit")
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
		ginkgo.By("Waiting for registry service node port")
		gomega.Eventually(func() (*corev1.Service, error) {
			service, err := getRegistryService(ctx, kubeClient, ns)
			if err != nil {
				return nil, err
			}

			if service != nil {
				registryPort := registry.GetServicePort(service)
				if registryPort.NodePort != 0 {
					registryHost = fmt.Sprintf("localhost:%d", registryPort.NodePort)
					return service, nil
				}
			}

			return nil, nil
		}, pollingDurationLong, pollingInterval).
			ShouldNot(gomega.BeNil())

		ginkgo.By("Checking registry for pushed image")
		gomega.Eventually(getImages(ctx, registryHost), pollingDurationLong, pollingInterval).
			Should(gomega.ContainElement("my-docker-username/helloworld"))

		ginkgo.By("Checking deployment container1")
		gomega.Eventually(selectContainerImage(kubeClient, ns, "app", "container1"), pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking deployment container2")
		gomega.Eventually(selectContainerImage(kubeClient, ns, "app", "container2"), pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking deployment container3")
		gomega.Eventually(selectContainerImage(kubeClient, ns, "app", "container3"), pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("should use local registry with kubectl deployment", func() {
		tempDir, err := framework.CopyToTempDir("tests/localregistry/testdata/local-registry-kubectl")
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
		gomega.Eventually(selectContainerImage(kubeClient, ns, "app", "container1"), pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking deployment container2")
		gomega.Eventually(selectContainerImage(kubeClient, ns, "app", "container2"), pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		ginkgo.By("Checking deployment container3")
		gomega.Eventually(selectContainerImage(kubeClient, ns, "app", "container3"), pollingDurationLong, pollingInterval).
			Should(gomega.MatchRegexp(`^localhost`))

		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("should use local registry with storage", func() {
		tempDir, err := framework.CopyToTempDir("tests/localregistry/testdata/local-registry-storage")
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

		ginkgo.By("Checking for registry statefulset")
		var actual *appsv1.StatefulSet
		gomega.Eventually(func() (*appsv1.StatefulSet, error) {
			var err error
			actual, err = kubeClient.RawClient().AppsV1().StatefulSets(ns).Get(context.TODO(), "registry-storage", metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return nil, nil
				}

				return nil, err
			}

			return actual, nil
		}, pollingDurationLong, pollingInterval).
			ShouldNot(gomega.BeNil())

		gomega.Expect(actual.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage]).
			To(gomega.Equal(resource.MustParse("5Gi")))

		err = <-done
		framework.ExpectNoError(err)
	})

	ginkgo.It("should update devImage with local registry image", func() {
		tempDir, err := framework.CopyToTempDir("tests/localregistry/testdata/local-registry-devimage")
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

		ginkgo.By("Checking for replaced deployment")
		var actuals *appsv1.DeploymentList
		gomega.Eventually(func() ([]appsv1.Deployment, error) {
			actuals, err = kubeClient.RawClient().AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: "devspace.sh/replaced=true"})
			if err != nil {
				return nil, err
			}

			return actuals.Items, nil
		}, pollingDurationLong, pollingInterval).
			ShouldNot(gomega.BeEmpty())

		actual := actuals.Items[0]
		gomega.Expect(actual.Spec.Template.Spec.Containers[0].Image).To(gomega.MatchRegexp("localhost"))
		gomega.Expect(actual.Spec.Template.Spec.Containers[0].Image).To(gomega.MatchRegexp("my-docker-username/helloworld-dev"))

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
			Log:      logpkg.NewStreamLogger(output, output, logrus.DebugLevel),
		}
		err = buildCmd.RunDefault(f)
		framework.ExpectError(err)
		gomega.Expect(output.String()).To(
			gomega.ContainSubstring("unable to push image my-docker-username/helloworld-kaniko and only docker and buildkit builds support using a local registry"),
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

func selectContainerImage(kubeHelper *kube.KubeHelper, namespace, deployment, containerName string) func() (string, error) {
	return func() (string, error) {
		deployment, err := kubeHelper.RawClient().AppsV1().Deployments(namespace).Get(context.TODO(), deployment, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return "", nil
			}

			return "", err
		}

		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == containerName {
				return container.Image, nil
			}
		}

		return "", nil
	}
}

func getRegistryService(ctx context.Context, kubeHelper *kube.KubeHelper, namespace string) (*corev1.Service, error) {
	service, err := kubeHelper.RawClient().CoreV1().Services(namespace).Get(ctx, "registry", metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}
	return service, nil
}

func getImages(ctx context.Context, registryHost string) func() ([]string, error) {
	return func() ([]string, error) {
		registry, err := name.NewRegistry(registryHost)
		if err != nil {
			return nil, err
		}

		images, err := remote.Catalog(ctx, registry)
		if err != nil {
			return nil, err
		}

		return images, nil
	}

}
