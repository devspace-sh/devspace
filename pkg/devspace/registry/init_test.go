package registry

import (
	//"encoding/base64"
	"fmt"
	"testing"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

var logOutput string

type testLogger struct {
	log.DiscardLogger
}

func (t testLogger) Done(args ...interface{}) {
	logOutput = logOutput + "\nDone " + fmt.Sprint(args...)
}
func (t testLogger) Donef(format string, args ...interface{}) {
	logOutput = logOutput + "\nDone " + fmt.Sprintf(format, args...)
}
func (t testLogger) Error(args ...interface{}) {
	logOutput = logOutput + "\nError " + fmt.Sprint(args...)
}
func (t testLogger) Errorf(format string, args ...interface{}) {
	logOutput = logOutput + "\nError " + fmt.Sprintf(format, args...)
}
func (t testLogger) Info(args ...interface{}) {
	logOutput = logOutput + "\nInfo " + fmt.Sprint(args...)
}
func (t testLogger) Infof(format string, args ...interface{}) {
	logOutput = logOutput + "\nInfo " + fmt.Sprintf(format, args...)
}
func (t testLogger) StartWait(message string) {
	logOutput = logOutput + "\nStartWait " + message
}
func (t testLogger) StopWait() {
	logOutput = logOutput + "\nStopWait"
}

type createPullSecretTestCase struct {
	name string

	namespace       string
	serviceAccounts []string
	imagesInConfig  map[string]*latest.ImageConfig

	expectedLog string
	expectedErr string
}

func TestCreatePullSecrets(t *testing.T) {
	testCases := []createPullSecretTestCase{
		createPullSecretTestCase{
			name:            "One simple creation without default service account",
			namespace:       "testNS",
			serviceAccounts: []string{"someServiceAccount"},
			imagesInConfig: map[string]*latest.ImageConfig{
				"testimage": &latest.ImageConfig{
					CreatePullSecret: ptr.Bool(true),
					Image:            ptr.String("testimage"),
				},
			},
			expectedLog: `
StartWait Creating image pull secret for registry: 
StopWait
Error Couldn't find service account 'default' in namespace 'testNS': serviceaccounts "default" not found`,
		},
		createPullSecretTestCase{
			name:            "One simple creation with default service account",
			namespace:       "testNS",
			serviceAccounts: []string{"default"},
			imagesInConfig: map[string]*latest.ImageConfig{
				"testimage": &latest.ImageConfig{
					CreatePullSecret: ptr.Bool(true),
					Image:            ptr.String("testimage"),
				},
			},
			expectedLog: `
StartWait Creating image pull secret for registry: 
StopWait`,
		},
	}

	for _, testCase := range testCases {
		pullSecretNames = []string{}
		//Setting up kubeClient
		kubeClient := fake.NewSimpleClientset()
		_, err := kubeClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testCase.namespace,
			},
		})
		assert.NilError(t, err, "Error creating namespace in testCase %s", testCase.name)
		for _, serviceAccount := range testCase.serviceAccounts {
			_, err = kubeClient.CoreV1().ServiceAccounts(testCase.namespace).Create(&k8sv1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: serviceAccount,
				},
			})
			assert.NilError(t, err, "Error creating serviceAccount in testCase %s", testCase.name)
		}

		// Create fake devspace config
		testConfig := &latest.Config{
			Images:      &testCase.imagesInConfig,
			Deployments: &[]*latest.DeploymentConfig{},
			Cluster:     &latest.Cluster{Namespace: &testCase.namespace},
		}

		logOutput = ""

		//Unfortunately we can't fake dockerClients yet.
		err = CreatePullSecrets(testConfig, nil, kubeClient, &testLogger{})

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error creating pull secrets in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error creating pull secrets in testCase %s", testCase.name)
		}
		assert.Equal(t, testCase.expectedLog, logOutput, "Wrong log output in testCase %s", testCase.name)
	}

	//TODO: Fake a dockerClient to make this work
	/*secretNames := GetPullSecretNames()
	assert.Equal(t, 1, len(secretNames), "Wrong number of secret names after creating one secret.")
	assert.Equal(t, "devspace-auth-docker", secretNames[0], "Wrong saved sercet name")

	resultSecret , err := kubeClient.CoreV1().Secrets(namespace).Get(secretNames[0], metav1.GetOptions{})
	assert.Equal(t, "devspace-auth-docker", resultSecret.ObjectMeta.Name, "Saved secret has wrong name")
	assert.Equal(t, `{
			"auths": {
				"https://index.docker.io/v1/": {
					"auth": "` + base64.StdEncoding.EncodeToString([]byte("someuser:password")) + `",
					"email": "someuser@example.com"
				}
			}
		}`, string(resultSecret.Data[k8sv1.DockerConfigJsonKey]), "Saved secret has wrong data")*/

}
