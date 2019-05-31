package registry

import (
	//"encoding/base64"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	
	//"gotest.tools/assert"
)

func TestCreatePullSecrets(t *testing.T){
	pullSecretNames = []string{}

	namespace := "testNS"
	//Setting up kubeClient
	kubeClient := fake.NewSimpleClientset()
	_, err := kubeClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}
	_, err = kubeClient.CoreV1().ServiceAccounts(namespace).Create(&k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	})
	if err != nil {
		t.Fatalf("Error creating serviceAcount: %v", err)
	}

	// Create fake devspace config
	testConfig := &latest.Config{
		Images: &map[string]*latest.ImageConfig{
			"testimage": &latest.ImageConfig{
				CreatePullSecret: ptr.Bool(true),
				Image: ptr.String("testimage"),
			},
		},
		Deployments: &[]*latest.DeploymentConfig{},
	}
	
	//Unfortunately we can't fake dockerClients yet.
	err = CreatePullSecrets(testConfig, nil, kubeClient, log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating pullSecrets: %v", err)
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
