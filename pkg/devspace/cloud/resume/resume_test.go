package resume

import (
	"testing"

	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	fakekubeconfig "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"
)

type resumeSpaceTestCase struct {
	name string

	kubeContext string

	expectedErr string
}

// TODO: Test past the GetProvider-call
func TestResume(t *testing.T) {
	testCases := []resumeSpaceTestCase{
		resumeSpaceTestCase{
			name:        "Context not a space",
			kubeContext: "nospace",
		},
	}

	for _, testCase := range testCases {
		resumer := NewSpaceResumer(&fakekube.Client{
			Client:     fake.NewSimpleClientset(),
			KubeLoader: &fakekubeconfig.Loader{},
			Context:    testCase.kubeContext,
		}, log.NewFakeLogger())

		err := resumer.ResumeSpace(false)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
		}
	}
}
