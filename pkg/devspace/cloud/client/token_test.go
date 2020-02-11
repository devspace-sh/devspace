package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	testconfig "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"

	"gotest.tools/assert"
)

type getTokenTestCase struct {
	name string

	answers     []string
	keyBefore   string
	sendToken   string
	provider    string
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
			name:        "Valid token but provider not there",
			keyBefore:   "someKey",
			sendToken:   validToken,
			provider:    "DoesntExist",
			expectedErr: "token save: Couldn't find provider DoesntExist",
		},
		getTokenTestCase{
			name:          "Valid token successfully saved",
			keyBefore:     "someKey",
			sendToken:     validToken,
			expectedToken: validToken,
			provider:      "exists",
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
	loader := testconfig.NewLoader(&latest.Config{
		Providers: []*latest.Provider{
			&latest.Provider{
				Name: "exists",
			},
		},
	})

	client := &client{
		host:      "http://localhost:1234",
		accessKey: testCase.keyBefore,
		token:     testCase.tokenBefore,
		provider:  testCase.provider,
		loader:    loader,
	}

	token, err := client.GetToken()

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	assert.Equal(t, token, testCase.expectedToken, "Unexpected token in testCase %s", testCase.name)

}
