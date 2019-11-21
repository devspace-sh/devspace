package cloud

/*
import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

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
