package cloud

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	testutil "github.com/devspace-cloud/devspace/pkg/util/testing"
)

var testProvider = &provider{
	Provider: latest.Provider{},
	client:   testutil.NewCloudClient(),
	loader:   testutil.NewLoader(&latest.Config{}),
	log:      log.Discard,
}

type saveTestCase struct {
	name string

	provider provider

	expectedConfig *latest.Config
	expectedErr    error
}

func TestSave(t *testing.T) {
	testCases := []saveTestCase{
		saveTestCase{
			name: "Save new provider",
			provider: provider{
				Provider: latest.Provider{
					Name: "newProv",
				},
			},
		},
	}

	for _, testCase := range testCases {
		testSave(t, testCase)
	}
}

func testSave(t *testing.T, testCase saveTestCase) {

}
