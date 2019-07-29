package cloud

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

// LoginEndpoint is the cloud endpoint that will log you in
const LoginEndpoint = "/login?cli=true"

// LoginSuccessEndpoint is the url redirected to after successful login
const LoginSuccessEndpoint = "/login-success"

// TokenEndpoint is the endpoint where to get a token from
const TokenEndpoint = "/auth/token"

// GetToken returns a valid access token to the provider
func (p *Provider) GetToken() (string, error) {
	if p.Key == "" {
		return "", errors.New("Provider has no key specified")
	}
	if p.Token != "" && token.IsTokenValid(p.Token) {
		return p.Token, nil
	}

	resp, err := http.Get(p.Host + TokenEndpoint + "?key=" + p.Key)
	if err != nil {
		return "", errors.Wrap(err, "token request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "read request body")
	}

	p.Token = string(body)
	if token.IsTokenValid(p.Token) == false {
		return "", errors.New("Received invalid token from provider")
	}

	err = p.Save()
	if err != nil {
		return "", errors.Wrap(err, "token save")
	}

	return p.Token, nil
}

// ReLogin loggs the user in with the given key or via browser
func ReLogin(providerConfig *latest.Config, cloudProvider string, key *string, log log.Logger) error {
	// Let's check if we are logged in first
	p := config.GetProvider(providerConfig, cloudProvider)
	if p == nil {
		cloudProviders := ""
		for _, p := range providerConfig.Providers {
			cloudProviders += p.Name + " "
		}

		return fmt.Errorf("Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: %s", cloudProviders)
	}

	provider := &Provider{
		*p,
	}
	if key != nil {
		provider.Token = ""
		provider.Key = *key

		// Check if we got access
		_, err := provider.GetSpaces()
		if err != nil {
			return fmt.Errorf("Access denied for key %s: %v", *key, err)
		}
	} else {
		provider.Token = ""
		provider.Key = ""

		err := provider.Login(log)
		if err != nil {
			return errors.Wrap(err, "Login")
		}
	}

	log.Donef("Successfully logged into %s", provider.Name)

	// Login into registries
	err := provider.LoginIntoRegistries(log)
	if err != nil {
		log.Warnf("Error logging into docker registries: %v", err)
	}

	// Save config
	err = provider.Save()
	if err != nil {
		return err
	}

	return nil
}

// EnsureLoggedIn checks if the user is logged into a certain cloud provider and if not loggs the user in
func EnsureLoggedIn(providerConfig *latest.Config, cloudProvider string, log log.Logger) error {
	// Let's check if we are logged in first
	p := config.GetProvider(providerConfig, cloudProvider)
	if p == nil {
		cloudProviders := ""
		for _, p := range providerConfig.Providers {
			cloudProviders += p.Name + " "
		}

		return fmt.Errorf("Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: %s", cloudProviders)
	}

	provider := &Provider{
		*p,
	}
	if provider.Key == "" {
		provider.Token = ""

		err := provider.Login(log)
		if err != nil {
			return errors.Wrap(err, "ensure logged in")
		}

		log.Donef("Successfully logged into %s", provider.Name)

		// Login into registries
		err = provider.LoginIntoRegistries(log)
		if err != nil {
			log.Warnf("Error logging into docker registries: %v", err)
		}

		err = provider.Save()
		if err != nil {
			return err
		}
	}

	return nil
}

// Login logs the user into DevSpace Cloud
func (p *Provider) Login(log log.Logger) error {
	var (
		url        = p.Host + LoginEndpoint
		ctx        = context.Background()
		keyChannel = make(chan string)
	)

	log.Infof("If the browser does not open automatically please navigate to %s", url)

	log.StartWait("Logging into cloud provider...")
	defer log.StopWait()

	server := startServer(p.Host+LoginSuccessEndpoint, keyChannel)
	open.Start(url)

	key := <-keyChannel
	close(keyChannel)

	err := server.Shutdown(ctx)
	if err != nil {
		return err
	}

	p.Key = key
	return nil
}

func startServer(redirectURI string, keyChannel chan string) *http.Server {
	srv := &http.Server{Addr: ":25853"}

	http.HandleFunc("/key", func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["key"]
		if !ok || len(keys[0]) < 1 {
			log.Fatal("Bad request")
		}

		keyChannel <- keys[0]
		http.Redirect(w, r, redirectURI, http.StatusSeeOther)
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
}
