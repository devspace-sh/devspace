package cloud

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	config "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	fakeBrowser "github.com/devspace-cloud/devspace/pkg/util/browser/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/pkg/errors"
	"gotest.tools/assert"
)

type getTokenTestCase struct {
	name string

	answers     []string
	keyBefore   string
	sendToken   string
	statusCode  int
	tokenBefore string

	expectedToken string
	expectedErr   string
}

func TestGetToken(t *testing.T) {
	claimAsJSON, _ := json.Marshal(token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	})
	validEncodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(validEncodedClaim), "=") {
		validEncodedClaim = strings.TrimSuffix(validEncodedClaim, "=")
	}
	validToken := "." + validEncodedClaim + "."

	testCases := []getTokenTestCase{
		getTokenTestCase{
			name:        "No key",
			expectedErr: "Provider has no key specified",
		},
		getTokenTestCase{
			name:          "Valid token from the start",
			keyBefore:     "someKey",
			tokenBefore:   validToken,
			expectedToken: validToken,
		},
		getTokenTestCase{
			name:        "Bad status code",
			keyBefore:   "someKey",
			statusCode:  http.StatusServiceUnavailable,
			sendToken:   "someResponseBody",
			expectedErr: "Error retrieving token: Code 503 => someResponseBody. Try to relogin with 'devspace login'",
		},
		getTokenTestCase{
			name:        "Invalid token from provider",
			keyBefore:   "someKey",
			sendToken:   "invalid",
			expectedErr: "Received invalid token from provider",
		},
		getTokenTestCase{
			name:          "Valid token from provider",
			keyBefore:     "someKey",
			sendToken:     validToken,
			expectedToken: validToken,
		},
	}

	var currentToken string
	var currentStatusCode int

	srv := &http.Server{Addr: ":1234"}

	http.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(currentStatusCode)
		w.Write([]byte(currentToken))
	})

	go srv.ListenAndServe()
	defer srv.Shutdown(context.Background())

	for _, testCase := range testCases {
		currentToken = testCase.sendToken
		currentStatusCode = testCase.statusCode
		if currentStatusCode == 0 {
			currentStatusCode = http.StatusOK
		}

		testGetToken(t, testCase)
	}
}

func testGetToken(t *testing.T, testCase getTokenTestCase) {
	logger := log.NewFakeLogger()
	for _, answer := range testCase.answers {
		logger.Survey.SetNextAnswer(answer)
	}

	provider := &provider{
		Provider: latest.Provider{
			Key:   testCase.keyBefore,
			Token: testCase.tokenBefore,
			Host:  "http://localhost:1234",
		},
		log:    logger,
		loader: config.NewLoader(&latest.Config{}),
	}

	token, err := provider.GetToken()

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	assert.Equal(t, token, testCase.expectedToken, "Unexpected token in testCase %s", testCase.name)

}

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
		}, /*
			loginTestCase{
				name:        "Invalid login",
				expectedKey: "",
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
		log: logger,
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
