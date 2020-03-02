package kubectl

import (
	"context"
	"net/http"
	"testing"

	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type otherNameForClientConfig clientcmd.ClientConfig
type fakeClientConfig struct {
	otherNameForClientConfig
}

func (f *fakeClientConfig) ClientConfig() (*restclient.Config, error) {
	return &restclient.Config{}, nil
}

type genericRequestTestCase struct {
	name string

	options GenericRequestOptions

	expectedOutput string
	expectedErr    bool
}

func TestGenericRequest(t *testing.T) {
	testCases := []genericRequestTestCase{
		{
			name: "Invalid api version",
			options: GenericRequestOptions{
				APIVersion: "a",
			},
			expectedErr: true,
		},
		{
			name: "Request with name and namespace",
			options: GenericRequestOptions{
				APIVersion: "a/b",
				Name:       "aName",
				Namespace:  "aNS",
			},
			expectedOutput: "Response2",
		},
		{
			name: "Request with labelSelector",
			options: GenericRequestOptions{
				APIVersion:    "a/b",
				LabelSelector: "label: selector",
			},
			expectedOutput: "Response",
		},
	}

	srv := &http.Server{Addr: ":80"}

	http.HandleFunc("/apis/a/b", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Response"))
	})

	http.HandleFunc("/apis/a/b/namespaces/aNS/aName", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Response2"))
	})

	go srv.ListenAndServe()
	defer srv.Shutdown(context.Background())

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		client := &client{
			Client:       kubeClient,
			ClientConfig: &fakeClientConfig{},
		}

		out, err := client.GenericRequest(&testCase.options)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}
		assert.Equal(t, out, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
	}
}
