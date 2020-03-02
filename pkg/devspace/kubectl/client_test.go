package kubectl

import (
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	fakekubeloader "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type newClientFromContextTestCase struct {
	name string

	context       string
	namespace     string
	switchContext bool
	contexts      map[string]*clientcmdapi.Context

	expectedErr    bool
	expectedClient *client
}

func TestNewClientFromContext(t *testing.T) {
	testCases := []newClientFromContextTestCase{
		{
			name:        "context not there",
			context:     "notThere",
			expectedErr: true,
		},
	}

	for _, testCase := range testCases {
		kubeLoader := &fakekubeloader.Loader{
			RawConfig: &clientcmdapi.Config{
				Contexts: testCase.contexts,
			},
		}
		client, err := NewClientFromContext(testCase.context, testCase.namespace, testCase.switchContext, kubeLoader)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		clientAsYaml, err := yaml.Marshal(client)
		assert.NilError(t, err, "Error parsing client to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedClient)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(clientAsYaml), string(expectedAsYaml), "Unexpected client in testCase %s", testCase.name)
	}
}

type printWarningTestCase struct {
	name string

	generatedConfig *generated.Config
	noWarning       bool
	shouldWait      bool
	clientNamespace string
	clientContext   string

	expectedErr bool
}

func TestPrintWarning(t *testing.T) {
	testCases := []printWarningTestCase{
		{
			name: "Last context is different than current",
			generatedConfig: &generated.Config{
				ActiveProfile: "active",
				Profiles: map[string]*generated.CacheConfig{
					"active": &generated.CacheConfig{
						LastContext: &generated.LastContextConfig{
							Context: "someContext",
						},
					},
				},
			},
			shouldWait:      true,
			clientNamespace: metav1.NamespaceDefault,
		},
		{
			name: "Last namespace is different than current",
			generatedConfig: &generated.Config{
				ActiveProfile: "active",
				Profiles: map[string]*generated.CacheConfig{
					"active": &generated.CacheConfig{
						LastContext: &generated.LastContextConfig{
							Namespace: "someNs",
						},
					},
				},
			},
		},
	}

	second = 0
	defer func() { second = time.Second }()

	for _, testCase := range testCases {
		client := &client{
			namespace:      testCase.clientNamespace,
			currentContext: testCase.clientContext,
		}
		err := client.PrintWarning(testCase.generatedConfig, testCase.noWarning, testCase.shouldWait, &log.FakeLogger{Level: logrus.InfoLevel})

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}
	}
}
