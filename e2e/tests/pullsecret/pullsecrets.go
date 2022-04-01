package pullsecret

import (
	"context"
	"encoding/base64"
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/onsi/ginkgo"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sort"
)

var _ = DevSpaceDescribe("pullsecret", func() {
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

	ginkgo.It("should create pullsecret with user & password", func() {
		tempDir, err := framework.CopyToTempDir("tests/pullsecret/testdata/simple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("pullsecret")
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

		// check if secrets are created
		pullSecret, err := kubeClient.RawClient().CoreV1().Secrets(ns).Get(context.TODO(), "test-secret", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(pullSecret.Data), 1)
		registryAuthEncoded := base64.StdEncoding.EncodeToString([]byte("my-user:my-password"))
		pullSecretDataValue := []byte(`{
			"auths": {
				"ghcr.io": {
					"auth": "` + registryAuthEncoded + `",
					"email": "noreply@devspace.sh"
				}
			}
		}`)
		framework.ExpectEqual(string(pullSecret.Data[k8sv1.DockerConfigJsonKey]), string(pullSecretDataValue))

		pullSecret, err = kubeClient.RawClient().CoreV1().Secrets(ns).Get(context.TODO(), "devspace-auth-ghcr2-io", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(pullSecret.Data), 1)
		registryAuthEncoded = base64.StdEncoding.EncodeToString([]byte("my-user2:my-password2"))
		pullSecretDataValue = []byte(`{
			"auths": {
				"ghcr2.io": {
					"auth": "` + registryAuthEncoded + `",
					"email": "noreply@devspace.sh"
				}
			}
		}`)
		framework.ExpectEqual(string(pullSecret.Data[k8sv1.DockerConfigJsonKey]), string(pullSecretDataValue))

		serviceAccount, err := kubeClient.RawClient().CoreV1().ServiceAccounts(ns).Get(context.TODO(), "default", metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(serviceAccount.ImagePullSecrets), 2)
		sort.Slice(serviceAccount.ImagePullSecrets, func(i, j int) bool {
			return serviceAccount.ImagePullSecrets[i].Name < serviceAccount.ImagePullSecrets[j].Name
		})
		framework.ExpectEqual(serviceAccount.ImagePullSecrets[0].Name, "devspace-auth-ghcr2-io")
		framework.ExpectEqual(serviceAccount.ImagePullSecrets[1].Name, "test-secret")
	})
})
