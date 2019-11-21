package cloud

/*
import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

func TestGetToken(t *testing.T) {
	_, err := (&Provider{latest.Provider{}, log.GetInstance()}).GetToken()
	assert.Error(t, err, "Provider has no key specified")

	testClaim := token.ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	}
	claimAsJSON, _ := json.Marshal(testClaim)
	encodedToken := "." + base64.URLEncoding.EncodeToString(claimAsJSON) + "."
	provider := &Provider{
		latest.Provider{
			Key:   "someKey",
			Token: encodedToken,
		},
		log.Discard,
	}
	token, err := provider.GetToken()
	assert.NilError(t, err, "Error getting predefined token")
	assert.Equal(t, token, encodedToken, "Predefined valid token not returned from GetToken")

	provider.Token = ""
	_, err = (provider).GetToken()
	assert.Error(t, err, "token request: Get /auth/token?key=someKey: unsupported protocol scheme \"\"", "Wrong or no error when trying to reach an unreachable host")
}

func TestReLogin(t *testing.T) {
	err := ReLogin(&latest.Config{Providers: []*latest.Provider{&latest.Provider{Name: "someProvider"}}}, "Doesn'tExist", nil, &log.DiscardLogger{})
	assert.Error(t, err, "Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: someProvider ", "No or wrong error when trying to reloigin with a non-existent provider")

	err = ReLogin(&latest.Config{Providers: []*latest.Provider{&latest.Provider{Name: "someProvider"}}}, "someProvider", ptr.String(""), &log.DiscardLogger{})
	assert.Error(t, err, "Access denied for key : get token: Provider has no key specified", "No or wrong error when trying to reloigin with a key-less provider")
}

func TestEnsureLoggedIn(t *testing.T) {
	err := EnsureLoggedIn(&latest.Config{Providers: []*latest.Provider{&latest.Provider{Name: "someProvider"}}}, "Doesn'tExist", &log.DiscardLogger{})
	assert.Error(t, err, "Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: someProvider ", "No or wrong error when trying to reloigin with a non-existent provider")
}*/
