package cloud

import (
	"net/http"
	"testing"

	client "github.com/devspace-cloud/devspace/pkg/devspace/cloud/client/testing"
	fakeBrowser "github.com/devspace-cloud/devspace/pkg/util/browser/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/pkg/errors"
	"gotest.tools/assert"
)

type loginTestCase struct {
	name string

	browserShouldFail bool
	answers           []string
	sendKey           string

	expectedKey string
	expectedErr string
}

func TestLogin(t *testing.T) {
	testCases := []loginTestCase{
		loginTestCase{
			name:        "Login with browser",
			sendKey:     "myKey",
			expectedKey: "myKey",
		},
		/*loginTestCase{
			name:        "Invalid login",
			expectedKey: "",
		},
		loginTestCase{
			name:              "Login with question",
			browserShouldFail: true,
			answers:           []string{" mykey "},
			expectedKey:       "mykey",
		},*/
	}

	for _, testCase := range testCases {
		testLogin(t, testCase)
	}
}

func testLogin(t *testing.T, testCase loginTestCase) {
	logger := log.NewFakeLogger()
	for _, answer := range testCase.answers {
		logger.Survey.SetNextAnswer(answer)
	}

	provider := &provider{
		browser: &fakeBrowser.FakeBrowser{
			RunCallback: func(url string) error {
				if testCase.browserShouldFail {
					return errors.New("")
				}
				return nil
			},
		},
		client: &client.CloudClient{},
		log:    logger,
	}

	go func() {
		err := provider.Login()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		assert.Equal(t, provider.Key, testCase.expectedKey, "Unexpected key in testCase %s", testCase.name)
	}()

	for true {
		queryString := ""
		if testCase.sendKey != "" {
			queryString = "key=" + testCase.sendKey
		}
		_, err := http.Get("http://localhost:25853/key?" + queryString)
		if err != nil {
			break
		}
	}
}
