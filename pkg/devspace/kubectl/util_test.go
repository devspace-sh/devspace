package kubectl

import (
	"context"
	"io"
	"testing"

	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"gotest.tools/assert"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// minimalClient implements kubectl.Client with only the two methods that
// IsMinikubeKubernetes actually calls. All other methods panic if invoked.
type minimalClient struct {
	context      string
	clientConfig clientcmd.ClientConfig
}

func (c *minimalClient) CurrentContext() string                   { return c.context }
func (c *minimalClient) ClientConfig() clientcmd.ClientConfig    { return c.clientConfig }
func (c *minimalClient) KubeClient() kubernetes.Interface        { panic("not implemented") }
func (c *minimalClient) Namespace() string                       { panic("not implemented") }
func (c *minimalClient) RestConfig() *rest.Config                { panic("not implemented") }
func (c *minimalClient) KubeConfigLoader() kubeconfig.Loader     { panic("not implemented") }
func (c *minimalClient) IsInCluster() bool                       { panic("not implemented") }
func (c *minimalClient) CopyFromReader(_ context.Context, _ *k8sv1.Pod, _, _ string, _ io.Reader) error {
	panic("not implemented")
}
func (c *minimalClient) Copy(_ context.Context, _ *k8sv1.Pod, _, _, _ string, _ []string) error {
	panic("not implemented")
}
func (c *minimalClient) ExecStream(_ context.Context, _ *ExecStreamOptions) error {
	panic("not implemented")
}
func (c *minimalClient) ExecBuffered(_ context.Context, _ *k8sv1.Pod, _ string, _ []string, _ io.Reader) ([]byte, []byte, error) {
	panic("not implemented")
}
func (c *minimalClient) ExecBufferedCombined(_ context.Context, _ *k8sv1.Pod, _ string, _ []string, _ io.Reader) ([]byte, error) {
	panic("not implemented")
}
func (c *minimalClient) GenericRequest(_ context.Context, _ *GenericRequestOptions) (string, error) {
	panic("not implemented")
}
func (c *minimalClient) ReadLogs(_ context.Context, _, _, _ string, _ bool, _ *int64) (string, error) {
	panic("not implemented")
}
func (c *minimalClient) Logs(_ context.Context, _, _, _ string, _ bool, _ *int64, _ bool) (io.ReadCloser, error) {
	panic("not implemented")
}
func (c *minimalClient) EnsureNamespace(_ context.Context, _ string, _ interface{ Debug(args ...interface{}) }) error {
	panic("not implemented")
}

// makeClient builds a test Client whose current context points at a cluster
// with the given extensions map.
func makeClient(contextName string, extensions map[string]runtime.Object) *minimalClient {
	apiCfg := clientcmdapi.NewConfig()
	apiCfg.Clusters[contextName] = &clientcmdapi.Cluster{
		Server:     "https://example.test:6443",
		Extensions: extensions,
	}
	apiCfg.Contexts[contextName] = &clientcmdapi.Context{
		Cluster: contextName,
	}
	apiCfg.CurrentContext = contextName

	cfg := clientcmd.NewNonInteractiveClientConfig(
		*apiCfg,
		contextName,
		&clientcmd.ConfigOverrides{},
		nil,
	)
	return &minimalClient{context: contextName, clientConfig: cfg}
}

func TestIsMinikubeKubernetes(t *testing.T) {
	t.Run("nil client returns false", func(t *testing.T) {
		assert.Equal(t, false, IsMinikubeKubernetes(nil))
	})

	t.Run("nil ClientConfig returns false", func(t *testing.T) {
		c := &minimalClient{context: "some-cluster", clientConfig: nil}
		assert.Equal(t, false, IsMinikubeKubernetes(c))
	})

	t.Run("context named 'minikube' returns true", func(t *testing.T) {
		c := makeClient(minikubeContext, nil)
		assert.Equal(t, true, IsMinikubeKubernetes(c))
	})

	t.Run("non-minikube context with no extensions returns false", func(t *testing.T) {
		c := makeClient("my-cluster", nil)
		assert.Equal(t, false, IsMinikubeKubernetes(c))
	})

	t.Run("cluster extension with minikube provider returns true", func(t *testing.T) {
		ext := &runtime.Unknown{
			Raw:         []byte(`{"provider":"minikube.sigs.k8s.io"}`),
			ContentType: runtime.ContentTypeJSON,
		}
		c := makeClient("my-cluster", map[string]runtime.Object{minikubeProvider: ext})
		assert.Equal(t, true, IsMinikubeKubernetes(c))
	})

	t.Run("cluster extension with different provider returns false", func(t *testing.T) {
		ext := &runtime.Unknown{
			Raw:         []byte(`{"provider":"some-other-provider"}`),
			ContentType: runtime.ContentTypeJSON,
		}
		c := makeClient("my-cluster", map[string]runtime.Object{"some-other-provider": ext})
		assert.Equal(t, false, IsMinikubeKubernetes(c))
	})

	// Some tools (e.g. Teleport) serialise kubeconfig extensions as plain YAML
	// strings rather than structured objects. runtime.ToUnstructured panics on
	// these via reflection instead of returning an error. isMinikubeExtension
	// must recover gracefully and return false.
	t.Run("string-valued extension does not panic and returns false", func(t *testing.T) {
		ext := &runtime.Unknown{
			Raw:         []byte(`"my-cluster-name"`), // bare JSON string, not an object
			ContentType: runtime.ContentTypeJSON,
		}
		c := makeClient("my-cluster", map[string]runtime.Object{"example.dev/cluster-name": ext})
		assert.Equal(t, false, IsMinikubeKubernetes(c))
	})
}
