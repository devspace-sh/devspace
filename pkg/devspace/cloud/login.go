package cloud

import (
	"context"
	"fmt"
	"net/http"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
)

// LoginEndpoint is the cloud endpoint that will log you in
const LoginEndpoint = "/login"

// LoginSuccessEndpoint is the url redirected to after successful login
const LoginSuccessEndpoint = "/login-success"

// ReLogin loggs the user in with the given token or via browser
func ReLogin(providerConfig ProviderConfig, cloudProvider string, token *string, log log.Logger) error {
	// Let's check if we are logged in first
	provider, ok := providerConfig[cloudProvider]
	if ok == false {
		cloudProviders := ""
		for name := range providerConfig {
			cloudProviders += name + " "
		}

		return fmt.Errorf("Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: %s", cloudProviders)
	}

	if token != nil {
		provider.Token = *token

		// Check if we got access
		_, err := provider.GetSpaces()
		if err != nil {
			return fmt.Errorf("Access denied for token %s: %v", *token, err)
		}
	} else {
		provider.Token = ""

		err := provider.Login(log)
		if err != nil {
			return errors.Wrap(err, "Login")
		}
	}

	log.Donef("Successfully logged into %s", provider.Name)

	// Login into registries
	err := provider.LoginIntoRegistries()
	if err != nil {
		log.Warnf("Error logging into docker registries: %v", err)
	}

	// Save config
	err = SaveCloudConfig(providerConfig)
	if err != nil {
		return err
	}

	return nil
}

// EnsureLoggedIn checks if the user is logged into a certain cloud provider and if not loggs the user in
func EnsureLoggedIn(providerConfig ProviderConfig, cloudProvider string, log log.Logger) error {
	// Let's check if we are logged in first
	provider, ok := providerConfig[cloudProvider]
	if ok == false {
		cloudProviders := ""
		for name := range providerConfig {
			cloudProviders += name + " "
		}

		return fmt.Errorf("Cloud provider not found! Did you run `devspace add provider [url]`? Existing cloud providers: %s", cloudProviders)
	}

	if provider.Token == "" {
		err := provider.Login(log)
		if err != nil {
			return errors.Wrap(err, "ensure logged in")
		}

		log.Donef("Successfully logged into %s", provider.Name)

		// Login into registries
		err = provider.LoginIntoRegistries()
		if err != nil {
			log.Warnf("Error logging into docker registries: %v", err)
		}

		err = SaveCloudConfig(providerConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

// Login logs the user into the devspace cloud
func (p *Provider) Login(log log.Logger) error {
	log.StartWait("Logging into cloud provider...")
	defer log.StopWait()

	ctx := context.Background()
	tokenChannel := make(chan string)

	server := startServer(p.Host+LoginSuccessEndpoint, tokenChannel)
	open.Start(p.Host + LoginEndpoint)

	token := <-tokenChannel
	close(tokenChannel)

	err := server.Shutdown(ctx)
	if err != nil {
		return err
	}

	p.Token = token
	return nil
}

func startServer(redirectURI string, tokenChannel chan string) *http.Server {
	srv := &http.Server{Addr: ":25853"}

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["token"]
		if !ok || len(keys[0]) < 1 {
			log.Fatal("Bad request")
		}

		tokenChannel <- keys[0]
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
