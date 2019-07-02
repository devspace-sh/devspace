package cloud

import (
	"encoding/json"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"

	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"gotest.tools/assert"
)

type createIngressTestCase struct {
	name string

	createServices           []simplifiedService
	ManagerCreateIngressPath bool
	doFakeGraphQLClient      bool
	serviceAnswer            string

	expectedErr string
}

type simplifiedService struct {
	name            string
	specExists      bool
	activeWithPorts bool
}

func (s simplifiedService) toService() *v1.Service {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.name,
		},
	}
	if s.specExists {
		service.Spec = v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
		}
		if s.activeWithPorts {
			service.Spec.Ports = []v1.ServicePort{
				v1.ServicePort{
					Port: 1,
				},
			}
		} else {
			service.Spec.ClusterIP = "None"
		}
	}
	return service
}

func TestCreateIngress(t *testing.T) {
	namespace := "testNS"
	testCases := []createIngressTestCase{
		createIngressTestCase{
			name: "Two inactive services",
			createServices: []simplifiedService{
				simplifiedService{
					name: "tiller-deploy",
				},
				simplifiedService{
					name:       "NoClusterIP",
					specExists: true,
				},
			},
			expectedErr: "Couldn't find any active services an ingress could connect to. Please make sure you have a service for your application",
		},
		createIngressTestCase{
			name: "No token",
			createServices: []simplifiedService{
				simplifiedService{
					name:            "active",
					specExists:      true,
					activeWithPorts: true,
				},
			},
			expectedErr: "graphql create ingress path: get token: Provider has no key specified",
		},
		createIngressTestCase{
			name: "Wrong result",
			createServices: []simplifiedService{
				simplifiedService{
					name:            "active",
					specExists:      true,
					activeWithPorts: true,
				},
			},
			ManagerCreateIngressPath: false,
			doFakeGraphQLClient:      true,
			expectedErr:              "Mutation returned wrong result",
		},
		createIngressTestCase{
			name: "Successful creation",
			createServices: []simplifiedService{
				simplifiedService{
					name:            "active",
					specExists:      true,
					activeWithPorts: true,
				},
				simplifiedService{
					name:            "otheractive",
					specExists:      true,
					activeWithPorts: true,
				}},
			serviceAnswer:            ":",
			ManagerCreateIngressPath: true,
			doFakeGraphQLClient:      true,
		},
	}

	for _, testCase := range testCases {
		provider := Provider{}
		kubeClient := fake.NewSimpleClientset()
		testConfig := &latest.Config{
			Cluster: &latest.Cluster{
				Namespace: ptr.String(namespace),
			},
		}
		for _, service := range testCase.createServices {
			kubeClient.CoreV1().Services(namespace).Create(service.toService())
		}

		if testCase.doFakeGraphQLClient {
			graphQLResponse := struct {
				ManagerCreateIngressPath bool `json:"manager_createKubeContextDomainIngressPath"`
			}{ManagerCreateIngressPath: testCase.ManagerCreateIngressPath}
			response, err := json.Marshal(graphQLResponse)
			assert.NilError(t, err, "Error parsing fake response in testCase %s", testCase.name)
			defaultGraphlClient = &fakeGraphQLClient{
				responsesAsJSON: []string{string(response)},
			}
		}
		if testCase.serviceAnswer != "" {
			survey.SetNextAnswer(testCase.serviceAnswer)
		}

		err := provider.CreateIngress(testConfig, kubeClient, &Space{Cluster: &Cluster{}}, "")
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error calling graphqlRequest in testCase: %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error when trying to do a graphql request in testCase %s", testCase.name)
		}

		defaultGraphlClient = &graphlClient{}
	}
}
